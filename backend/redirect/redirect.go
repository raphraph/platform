package redirect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/syncloud/platform/config"
	"github.com/syncloud/platform/identification"
	"github.com/syncloud/platform/network"
	"github.com/syncloud/platform/version"
	"io"
	"log"
	"net/http"
)

type Service struct {
	UserPlatformConfig *config.UserConfig
	identification     *identification.Parser
}

func New(userPlatformConfig *config.UserConfig, identification *identification.Parser) *Service {
	return &Service{
		UserPlatformConfig: userPlatformConfig,
		identification:     identification,
	}
}

func (r *Service) Authenticate(email string, password string) (*User, error) {
	url := fmt.Sprintf("%s/user/get?email=%s&password=%s", r.UserPlatformConfig.GetRedirectApiUrl(), email, password)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = CheckHttpError(resp.StatusCode, body)
	if err != nil {
		return nil, err
	}
	var redirectUserResponse UserResponse
	err = json.Unmarshal(body, &redirectUserResponse)
	if err != nil {
		return nil, err
	}
	return &redirectUserResponse.Data, nil
}

func (r *Service) Acquire(email string, password string, domain string) (*Domain, error) {

	deviceId, err := r.identification.Id()
	if err != nil {
		return nil, err
	}

	request := &FreeDomainAcquireRequest{
		Email:            email,
		Password:         password,
		Domain:           domain,
		DeviceMacAddress: deviceId.MacAddress,
		DeviceName:       deviceId.Name,
		DeviceTitle:      deviceId.Title}
	url := fmt.Sprintf("%s/%s", r.UserPlatformConfig.GetRedirectApiUrl(), "domain/acquire_v2")
	requestJson, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	responseJson, err := http.Post(url, "application/json", bytes.NewBuffer(requestJson))
	if err != nil {
		return nil, err
	}
	defer responseJson.Body.Close()
	body, err := io.ReadAll(responseJson.Body)
	if err != nil {
		return nil, err
	}
	err = CheckHttpError(responseJson.StatusCode, body)
	if err != nil {
		return nil, err
	}
	log.Printf("acquire response: %s", body)
	var response FreeDomainAcquireResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, fmt.Errorf("failed to acquire domain")
	}
	return response.Data, nil
}

func (r *Service) Reset(updateToken string) error {
	return r.Update(nil, nil, config.WebAccessPort, config.WebProtocol, updateToken, false)
}

func (r *Service) Update(externalIp *string, webPort *int, webLocalPort int, webProtocol string, updateToken string, externalAccess bool) error {

	platformVersion, err := version.PlatformVersion()
	if err != nil {
		return err
	}

	localIp, err := network.LocalIPv4()
	if err != nil {
		return err
	}

	request := &FreeDomainUpdateRequest{
		Token:           updateToken,
		PlatformVersion: platformVersion,
		LocalIp:         localIp.String(),
		MapLocalAddress: !externalAccess,
		WebProtocol:     webProtocol,
		WebPort:         webPort,
		WebLocalPort:    webLocalPort,
	}

	if externalIp == nil {
		externalIp, err := network.PublicIPv4()
		if err != nil {
			return err
		}
		log.Printf("getting external ip: %s", externalIp)
	}

	if externalAccess {
		request.Ip = externalIp
	}

	ipv6Addr, err := network.IPv6()
	if err == nil {
		ipv6 := ipv6Addr.String()
		request.Ipv6 = &ipv6
	}

	dkimKey := r.UserPlatformConfig.GetDkimKey()
	if dkimKey != nil {
		request.DkimKey = dkimKey
	}

	url := fmt.Sprintf("%s/%s", r.UserPlatformConfig.GetRedirectApiUrl(), "domain/update")

	log.Printf("url: %s", url)
	requestJson, err := json.Marshal(request)
	log.Printf("request: %s", requestJson)

	responseJson, err := http.Post(url, "application/json", bytes.NewBuffer(requestJson))
	if err != nil {
		return err
	}
	defer responseJson.Body.Close()
	body, err := io.ReadAll(responseJson.Body)
	if err != nil {
		return err
	}
	err = CheckHttpError(responseJson.StatusCode, body)
	if err != nil {
		return err
	}

	return nil
}

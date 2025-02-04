package redirect

import (
	"encoding/json"
	"github.com/syncloud/platform/util"
	"log"
)

func CheckHttpError(status int, body []byte) error {
	if status == 200 {
		return nil
	}
	var redirectResponse Response
	err := json.Unmarshal(body, &redirectResponse)
	bodyString := string(body)
	if err != nil {
		log.Printf("error parsing redirect response: %v\n", err)
		return &util.PassThroughJsonError{
			Message: "Unable to parse Redirect response",
			Json:    bodyString,
		}
	}
	log.Printf("http error: %s\n", bodyString)
	return &util.PassThroughJsonError{
		Message: redirectResponse.Message,
		Json:    bodyString,
	}
}

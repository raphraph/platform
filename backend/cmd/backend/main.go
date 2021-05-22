package main

import (
	"github.com/spf13/cobra"
	"github.com/syncloud/platform/activation"
	"github.com/syncloud/platform/auth"
	"github.com/syncloud/platform/certificate"
	"github.com/syncloud/platform/config"
	"github.com/syncloud/platform/connection"
	"github.com/syncloud/platform/cron"
	"github.com/syncloud/platform/event"
	"github.com/syncloud/platform/identification"
	"github.com/syncloud/platform/nginx"
	"github.com/syncloud/platform/redirect"
	"github.com/syncloud/platform/snap"
	"github.com/syncloud/platform/systemd"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/syncloud/platform/backup"
	"github.com/syncloud/platform/installer"
	"github.com/syncloud/platform/job"
	"github.com/syncloud/platform/rest"
	"github.com/syncloud/platform/storage"
)

func main() {

	var rootCmd = &cobra.Command{Use: "backend"}
	configDb := rootCmd.PersistentFlags().String("config", config.DefaultConfigDb, "sqlite config db")
	redirectDomain := rootCmd.PersistentFlags().String("redirect-domain", "syncloud.it", "redirect domain")
	redirectUrl := rootCmd.PersistentFlags().String("redirect-url", "https://api.syncloud.it", "redirect url")
	idConfig := rootCmd.PersistentFlags().String("identification-config", "/etc/syncloud/id.cfg", "id config")

	var tcpCmd = &cobra.Command{
		Use:   "tcp [address]",
		Short: "listen on a tcp address, like localhost:8080",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			backend, err := Backend(*configDb, *redirectDomain, *redirectUrl, *idConfig)
			if err != nil {
				log.Print("error: ", err)
				os.Exit(1)
			}
			backend.Start("tcp", args[0])
		},
	}

	var unixSocketCmd = &cobra.Command{
		Use:   "unix [address]",
		Args:  cobra.ExactArgs(1),
		Short: "listen on a unix socket, like /tmp/backend.sock",
		Run: func(cmd *cobra.Command, args []string) {
			_ = os.Remove(args[0])
			backend, err := Backend(*configDb, *redirectDomain, *redirectUrl, *idConfig)
			if err != nil {
				log.Print("error: ", err)
				os.Exit(1)
			}
			backend.Start("unix", args[0])
		},
	}

	rootCmd.AddCommand(tcpCmd, unixSocketCmd)
	if err := rootCmd.Execute(); err != nil {
		log.Print("error: ", err)
		os.Exit(1)
	}
}

func Backend(configDb string, redirectDomain string, defaultRedirectUrl string, idConfig string) (*rest.Backend, error) {

	cronService := cron.New(cron.Job, time.Minute*5)
	cronService.Start()

	master := job.NewMaster()
	backupService := backup.NewDefault()
	eventTrigger := event.New()
	installerService := installer.New()
	storageService := storage.New()
	userConfig, err := config.NewUserConfig(configDb, config.OldConfig, redirectDomain, defaultRedirectUrl)
	if err != nil {
		return nil, err
	}
	redirectApiUrl := userConfig.GetRedirectApiUrl()
	redirectUrl, err := url.Parse(redirectApiUrl)
	if err != nil {
		return nil, err
	}

	id := identification.New(idConfig)
	redirectService := redirect.New(userConfig, id)
	worker := job.NewWorker(master)
	systemConfig, err := config.NewSystemConfig(config.File)
	if err != nil {
		return nil, err
	}

	snapService := snap.NewService()
	dataDir, err := systemConfig.DataDir()
	if err != nil {
		return nil, err
	}
	appDir, err := systemConfig.AppDir()
	if err != nil {
		return nil, err
	}
	configDir, err := systemConfig.ConfigDir()
	if err != nil {
		return nil, err
	}
	ldapService := auth.New(snapService, *dataDir, *appDir, *configDir)
	nginxService := nginx.New(systemd.New(), systemConfig, userConfig)
	freeActivation := activation.New(&connection.Internet{}, userConfig, redirectService, certificate.New(), ldapService, nginxService, eventTrigger)

	return rest.NewBackend(master, backupService, eventTrigger, worker, redirectService, installerService, storageService, redirectUrl, id, freeActivation), nil

}

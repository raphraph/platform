package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/syncloud/platform/event"
	"github.com/syncloud/platform/identification"
	"github.com/syncloud/platform/installer"
	"github.com/syncloud/platform/redirect"
	"github.com/syncloud/platform/rest/model"
	"github.com/syncloud/platform/storage"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/syncloud/platform/backup"
	"github.com/syncloud/platform/job"
)

type Backend struct {
	Master         *job.Master
	backup         *backup.Backup
	eventTrigger   *event.Trigger
	worker         *job.Worker
	redirect       *redirect.Redirect
	installer      installer.AppInstaller
	storage        *storage.Storage
	redirectProxy  *httputil.ReverseProxy
	identification *identification.Parser
}

func NewBackend(master *job.Master, backup *backup.Backup,
	eventTrigger *event.Trigger, worker *job.Worker,
	redirect *redirect.Redirect, installerService *installer.Installer,
	storageService *storage.Storage, redirectUrl *url.URL,
	identification *identification.Parser) *Backend {

	return &Backend{
		Master:         master,
		backup:         backup,
		eventTrigger:   eventTrigger,
		worker:         worker,
		redirect:       redirect,
		installer:      installerService,
		storage:        storageService,
		redirectProxy:  httputil.NewSingleHostReverseProxy(redirectUrl),
		identification: identification,
	}
}

func (backend *Backend) Start(network string, address string) {
	listener, err := net.Listen(network, address)
	if err != nil {
		panic(err)
	}

	go backend.worker.Start()

	r := mux.NewRouter()
	r.HandleFunc("/job/status", Handle(backend.JobStatus)).Methods("GET")
	r.HandleFunc("/backup/list", Handle(backend.BackupList)).Methods("GET")
	r.HandleFunc("/backup/create", Handle(backend.BackupCreate)).Methods("POST")
	r.HandleFunc("/backup/restore", Handle(backend.BackupRestore)).Methods("POST")
	r.HandleFunc("/backup/remove", Handle(backend.BackupRemove)).Methods("POST")
	r.HandleFunc("/installer/upgrade", Handle(backend.InstallerUpgrade)).Methods("POST")
	r.HandleFunc("/storage/disk_format", Handle(backend.StorageFormat)).Methods("POST")
	r.HandleFunc("/storage/boot_extend", Handle(backend.StorageBootExtend)).Methods("POST")
	r.HandleFunc("/event/trigger", Handle(backend.EventTrigger)).Methods("POST")
	r.PathPrefix("/redirect").Handler(http.StripPrefix("/redirect", backend.redirectProxy))
	r.HandleFunc("/id", Handle(backend.Id)).Methods("GET")

	r.Use(middleware)

	server := http.Server{}

	log.Println("Started backend")
	_ = server.Serve(listener)

}

func fail(w http.ResponseWriter, err error) {
	log.Println("error: ", err)
	appError := err.Error()
	response := model.Response{
		Success: false,
		Message: &appError,
	}
	responseJson, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		_, _ = fmt.Fprintf(w, string(responseJson))
	}
}

func success(w http.ResponseWriter, data interface{}) {
	response := model.Response{
		Success: true,
		Data:    &data,
	}
	responseJson, err := json.Marshal(response)
	if err != nil {
		fail(w, err)
	} else {
		_, _ = fmt.Fprintf(w, string(responseJson))
	}
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func Handle(f func(req *http.Request) (interface{}, error)) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		log.Printf("request: %s\n", req.URL.Path)
		data, err := f(req)
		if err != nil {
			fail(w, err)
		} else {
			success(w, data)
		}
	}
}

func (backend *Backend) BackupList(_ *http.Request) (interface{}, error) {
	return backend.backup.List()
}

func (backend *Backend) BackupRemove(req *http.Request) (interface{}, error) {
	var request model.BackupRemoveRequest
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		log.Printf("parse error: %v", err.Error())
		return nil, errors.New("file is missing")
	}
	err = backend.backup.Remove(request.File)
	if err != nil {
		return nil, err
	}
	return "removed", nil
}

func (backend *Backend) BackupCreate(req *http.Request) (interface{}, error) {
	var request model.BackupCreateRequest
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		log.Printf("parse error: %v", err.Error())
		return nil, errors.New("app is missing")
	}
	_ = backend.Master.Offer(func() { backend.backup.Create(request.App) })
	return "submitted", nil
}

func (backend *Backend) BackupRestore(req *http.Request) (interface{}, error) {
	var request model.BackupRestoreRequest
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		log.Printf("parse error: %v", err.Error())
		return nil, errors.New("file is missing")
	}
	_ = backend.Master.Offer(func() { backend.backup.Restore(request.File) })
	return "submitted", nil
}

func (backend *Backend) InstallerUpgrade(_ *http.Request) (interface{}, error) {
	_ = backend.Master.Offer(func() { backend.installer.Upgrade() })
	return "submitted", nil
}

func (backend *Backend) JobStatus(_ *http.Request) (interface{}, error) {
	return backend.Master.Status().String(), nil
}

func (backend *Backend) StorageFormat(req *http.Request) (interface{}, error) {
	var request model.StorageFormatRequest
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		log.Printf("parse error: %v", err.Error())
		return nil, errors.New("device is missing")
	}
	_ = backend.Master.Offer(func() { backend.storage.Format(request.Device) })
	return "submitted", nil
}

func (backend *Backend) EventTrigger(req *http.Request) (interface{}, error) {
	log.Printf("event trigger")
	var request model.EventTriggerRequest
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		log.Printf("parse error: %v", err.Error())
		return nil, errors.New("event is missing")
	}
	return "ok", backend.eventTrigger.RunEventOnAllApps(request.Event)
}

func (backend *Backend) Id(_ *http.Request) (interface{}, error) {
	id, err := backend.identification.Id()
	if err != nil {
		log.Printf("parse error: %v", err.Error())
		return nil, errors.New("id is not available")
	}
	return id, nil
}

func (backend *Backend) StorageBootExtend(_ *http.Request) (interface{}, error) {
	_ = backend.Master.Offer(func() { backend.storage.BootExtend() })
	return "submitted", nil
}

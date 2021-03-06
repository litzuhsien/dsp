package main

import (
	"fmt"
	"github.com/clixxa/dsp/dsp_flights"
	"github.com/clixxa/dsp/services"
	"github.com/clixxa/dsp/wish_flights"
	"log"
	"net/http"
	"os"
)

type Main struct {
	TestOnly bool
}

func (m *Main) Launch() {
	consul := &services.ConsulConfigs{}
	deps := &services.ProductionDepsService{Consul: consul}

	dspRuntime := &dsp_flights.BidEntrypoint{AllTest: m.TestOnly, Logic: dsp_flights.SimpleLogic{}}
	winRuntime := &wish_flights.WishEntrypoint{}

	router := &services.RouterService{}
	router.Mux = http.NewServeMux()
	router.Mux.Handle("/", dspRuntime)
	router.Mux.Handle("/win", winRuntime)

	cycler := &services.CycleService{}
	cycler.BindingDeps.Logger = log.New(os.Stdout, "INIT ", log.Lshortfile|log.Ltime)

	launch := &services.LaunchService{}

	wireUp := &services.CycleService{Proxy: func() error {
		dspRuntime.BindingDeps = deps.BindingDeps
		winRuntime.BindingDeps = deps.BindingDeps
		cycler.BindingDeps = deps.BindingDeps
		router.BindingDeps = deps.BindingDeps
		launch.BindingDeps = deps.BindingDeps
		return nil
	}}

	cycler.Children = append(cycler.Children, consul, deps, wireUp, dspRuntime, winRuntime)
	launch.Children = append(launch.Children, cycler, router)

	fmt.Println("starting launcher")
	launch.Launch()
}

func NewMain() *Main {
	m := &Main{}
	for _, flag := range os.Args[1:] {
		fmt.Printf(`arg %s`, flag)
		switch flag {
		case "test":
			m.TestOnly = true
		}
	}
	return m
}

func main() {
	NewMain().Launch()
}

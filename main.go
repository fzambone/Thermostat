package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/jfyne/live"
	"html/template"
	"io"
	"log"
	"net/http"
)

type ThermoModel struct {
	Name        string
	Temperature float32
	Status      string
}

func NewThermoModel(ctx context.Context, s live.Socket) *ThermoModel {
	m, ok := s.Assigns().(*ThermoModel)

	if !ok {
		m = &ThermoModel{
			Name:        live.Request(ctx).URL.Query().Get("name"),
			Temperature: 23.1,
			Status:      "-",
		}
	}

	return m
}

func thermoMount(ctx context.Context, s live.Socket) (interface{}, error) {
	log.Println("Mounting application")
	return NewThermoModel(ctx, s), nil
}

func tempUp(ctx context.Context, s live.Socket, p live.Params) (interface{}, error) {
	model := NewThermoModel(ctx, s)
	model.Temperature += 0.1
	return model, nil
}

func tempDown(ctx context.Context, s live.Socket, p live.Params) (interface{}, error) {
	model := NewThermoModel(ctx, s)
	model.Temperature -= 0.1
	return model, nil
}

func tempChange(ctx context.Context, s live.Socket, p live.Params) (interface{}, error) {
	model := NewThermoModel(ctx, s)
	t0 := model.Temperature
	model.Temperature += p.Float32("temperature")

	//model.Status = fmt.Sprintf("Temperature changed form %f to %f", t0, model.Temperature)

	s.Broadcast("status", fmt.Sprintf(model.Name+": Temperature changed form %f to %f", t0, model.Temperature))

	return model, nil
}

func render(ctx context.Context, data *live.RenderContext) (io.Reader, error) {
	tmpl, err := template.New("thermo").Parse(`
		<html>
			<head>
				<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-T3c6CoIi6uLrA9TneNEoa7RxnatzjcDSCmG1MXxSR1GAsXEV/Dwwykc2MPK8M2HN" crossorigin="anonymous">
				<script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/js/bootstrap.bundle.min.js" integrity="sha384-C6RzsynM9kWDrMNeT87bh95OGNyZPhcTNXj1NW7RuBCsyN/o0jlpcV8Qyq46cDfL" crossorigin="anonymous"></script>
			</head>
			<body>
				<div class="container" style="text-align: center">
					<h4>User: {{.Assigns.Name}}</h4>
					<h2>Temperature: {{.Assigns.Temperature}}C</h2>
					<div>
						status: {{.Assigns.Status}}
					</div>
					<div style="padding-top: 20px">
						<button live-click="temp-up" class="btn btn-success btn-sm">+0.1C</button> - 
						<button live-click="temp-down" class="btn btn-success btn-sm">-0.1C</button>
					</div>
					<div style="padding-top: 20px">
						<button live-click="temp-change" live-value-temperature="2" class="btn btn-success btn-sm">+2C</button> - 
						<button live-click="temp-change" live-value-temperature="-2" class="btn btn-success btn-sm">-2C</button>
					</div>
				</div>
			<!-- Include to make live work -->
			<script src="/live.js"></script>
			</body>
		</html>
	`)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return &buf, nil
}

func main() {
	fmt.Println("Application is starting ...")

	h := live.NewHandler()
	h.HandleRender(render)
	h.HandleMount(thermoMount)

	h.HandleEvent("temp-up", tempUp)
	h.HandleEvent("temp-down", tempDown)
	h.HandleEvent("temp-change", tempChange)
	h.HandleSelf("status", func(ctx context.Context, s live.Socket, data interface{}) (interface{}, error) {
		model := NewThermoModel(ctx, s)
		model.Status = data.(string)
		return model, nil
	})

	http.Handle("/thermostat", live.NewHttpHandler(live.NewCookieStore("session-name", []byte("weak-secret")), h))
	http.Handle("/live.js", live.Javascript{})
	http.ListenAndServe(":8080", nil)
}

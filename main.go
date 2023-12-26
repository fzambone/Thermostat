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
	"time"
)

type ThermoModel struct {
	Name        string
	Temperature float32
	Status      string
	Time        string
}

func NewThermoModel(ctx context.Context, s live.Socket) *ThermoModel {
	m, ok := s.Assigns().(*ThermoModel)

	if !ok {
		m = &ThermoModel{
			Name:        live.Request(ctx).URL.Query().Get("name"),
			Temperature: 17.5,
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

// saveEvent sends chat like event
func saveEvent(ctx context.Context, s live.Socket, p live.Params) (interface{}, error) {
	model := NewThermoModel(ctx, s)
	message := p.String("message")

	s.Broadcast("status", model.Name+": "+message)

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
						{{if gt .Assigns.Temperature 25.0}}
							<h4 style="color: red">Warning: Temperature is too high!!! (Over 25C)</h4>
						{{end}}
					</div>
					<div style="padding-top: 20px">
						<button live-click="temp-up" live-window-keyup="temp-up" live-key="ArrowUp" class="btn btn-success btn-sm">+0.1C</button> - 
						<button live-click="temp-down" live-window-keydown="temp-down" live-key="ArrowDown" class="btn btn-success btn-sm">-0.1C</button>
					</div>
					<div style="padding-top: 20px; padding-bottom: 20px">
						<button live-click="temp-change" live-value-temperature="2" class="btn btn-success btn-sm">+2C</button> - 
						<button live-click="temp-change" live-value-temperature="-2" class="btn btn-success btn-sm">-2C</button>
					</div>
					<div style="border: 1px solid black; padding: 5px">
						{{.Assigns.Time}}
					</div>
					<div style="padding: 10px">
						<form live-submit="save" live-hook="submit">
							<input type="text" name="message" />&#160;
							<input type="submit" value="send..." class="btn btn-success btn-sm" />
						</form>
					</div>
					<div live-update="prepend">
						{{.Assigns.Status}}
					</div>
				</div>
			<!-- Include to make live work -->
			<script src="/live.js"></script>
			<script>
				window.Hooks = {
					"submit": {
						mounted: function() {
							this.el.addEventListener("submit", () => {
								this.el.querySelector("input").value = "";
							});
						}
					}
				};
			</script>
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
	h.HandleEvent("save", saveEvent)
	h.HandleSelf("status", func(ctx context.Context, s live.Socket, data interface{}) (interface{}, error) {
		model := NewThermoModel(ctx, s)
		model.Status = data.(string)
		return model, nil
	})
	h.HandleSelf("time", func(ctx context.Context, s live.Socket, data interface{}) (interface{}, error) {
		model := NewThermoModel(ctx, s)
		model.Time = data.(string)
		return model, nil
	})

	lh := live.NewHttpHandler(live.NewCookieStore("session-name", []byte("weak-secret")), h)
	go func() {
		for {
			lh.Broadcast("time", time.Now().Format(time.RFC1123))
			time.Sleep(1 * time.Second)
		}
	}()

	http.Handle("/thermostat", lh)
	http.Handle("/live.js", live.Javascript{})
	http.ListenAndServe(":8080", nil)
}

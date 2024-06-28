package main

import (
	logger "github.com/chi-middleware/logrus-logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

var (
	args = map[string]operation{
		"mtr": {
			Command: "mtr",
			Args:    []string{"-c", "5", "-r", "-w", "-b"},
			Rules:   []string{"ip", "hostname"},
		},
		"traceroute": {
			Command: "traceroute",
			Args:    []string{"-w", "1", "-q", "1"},
			Rules:   []string{"ip", "hostname"},
		},
		"ping": {
			Command: "ping",
			Args:    []string{"-c", "{viper:feature.ping.count}"},
			Rules:   []string{"ip", "hostname"},
		},
		"bgp": {
			Command: "birdc",
			Args:    []string{"-r", "sh", "ro", "all", "for"},
			Rules:   []string{"ip", "cidr"},
		},
	}

	v = validator.New()
)

type operation struct {
	// Command is the command to execute
	Command string
	// Args are the command args - the argument "{target}" will be replaced
	Args []string
	// Rules are the validation rules. At least ONE rule must match.
	Rules []string
}

type request struct {
	Type   string `json:"type"`
	Target string `json:"target"`
}

type response struct {
	Error string `json:"error,omitempty"`
	Data  string `json:"data,omitempty"`
}

func main() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetDefault("rate.limit", 30)
	viper.SetDefault("rate.timeframe", 1*time.Minute)

	// Common features, enabled by default
	viper.SetDefault("feature.mtr", true)
	viper.SetDefault("feature.traceroute", true)
	viper.SetDefault("feature.ping", true)
	viper.SetDefault("feature.ping.count", "5")

	// Extra features, disabled by default
	viper.SetDefault("feature.files", false)
	viper.SetDefault("feature.files.path", "/data/files")
	viper.SetDefault("feature.bgp", false)

	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(httprate.LimitByIP(viper.GetInt("rate.limit"), viper.GetDuration("rate.timeframe")))

	allowedOrigins := []string{"*"}

	if origin := viper.GetString("cors.origin"); origin != "" {
		allowedOrigins = []string{origin}
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "X-Requested-With"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	l := log.New()
	r.Use(logger.Logger("router", l))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, map[string]string{
			"message": "https://github.com/ccatss/smoked",
		})
	})

	if viper.GetBool("feature.files") {
		log.WithField("path", viper.GetString("feature.files.path")).Info("Enabling file server")

		fs := http.FileServer(http.Dir(viper.GetString("feature.files.path")))
		fs = http.StripPrefix("/files", fs)

		r.Get("/files/*", func(w http.ResponseWriter, r *http.Request) {
			fs.ServeHTTP(w, r)
		})
	}

	r.Post("/lg", handleRequest)

	log.Info("Starting server...")

	err := http.ListenAndServe(":8080", r)

	if err != nil {
		log.WithError(err).Fatal("Unable to start server")
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	var req request

	if err := render.Decode(r, &req); err != nil {
		render.JSON(w, r, response{
			Error: err.Error(),
		})
		return
	}

	arg, ok := args[req.Type]

	if !ok {
		render.JSON(w, r, response{
			Error: "Invalid type",
		})
		return
	}

	validated := false

	for _, rule := range arg.Rules {
		if err := v.Var(req.Target, rule); err == nil {
			validated = true
			break
		}
	}

	if !validated {
		render.JSON(w, r, response{
			Error: "Invalid target",
		})
		return
	}

	if b, err := arg.Execute(req.Target); err != nil {
		render.JSON(w, r, response{
			Error: err.Error(),
		})
	} else {
		render.JSON(w, r, response{
			Data: string(b),
		})
	}
}

func handleFile(w http.ResponseWriter, r *http.Request) {

}

func (o operation) Enabled() bool {
	return viper.GetBool("feature." + o.Command)
}

func (o operation) Execute(target string) ([]byte, error) {
	args := make([]string, len(o.Args))
	copy(args, o.Args)

	// Check if {target} is present for replacement
	// This isn't currently used, but future features could use it if necessary
	var found bool

	for i, arg := range args {
		switch {
		case arg == "{target}":
			args[i] = target
			found = true
		case strings.HasPrefix(arg, "{viper:") && strings.HasSuffix(arg, "}"):
			args[i] = viper.GetString(arg[7 : len(arg)-1])
		}
	}

	if !found {
		args = append(args, target)
	}

	cmd := exec.Command(o.Command, args...)

	// This mimics the behavior of the current system.
	// It could be done as a stream and WAY better, but it works.
	b, err := cmd.CombinedOutput()

	if err != nil {
		return nil, err
	}

	return b, nil
}

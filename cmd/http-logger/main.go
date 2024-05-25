// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// pluginName is the plugin name
var pluginName = "http-logger"

// ClientRegisterer is the symbol the plugin loader will try to load. It must implement the RegisterClient interface
var ClientRegisterer = registerer(pluginName)

// HandlerRegisterer is the symbol the plugin loader will try to load. It must implement the Registerer interface
var HandlerRegisterer = registerer(pluginName)

type registerer string

const LAYOUT_FORMAT = "2006-01-02 15:04:05"
const LOG_FORMAT = "TIMESTAMP: %s | LOG-LEVEL: %s | USER-AGENT: %s | URL: %s | IP-CLIENT: %s | METHOD: %s | HTTP-CODE: %d | REQUEST-HEADER: %v | QUERY-PARAMS: %v | REQUEST-BODY: %s | RESPONSE: %s"

func (r registerer) RegisterHandlers(f func(
	name string,
	handler func(context.Context, map[string]interface{}, http.Handler) (http.Handler, error),
)) {
	f(string(r), r.registerHandlers)
}

func (r registerer) registerHandlers(_ context.Context, extra map[string]interface{}, h http.Handler) (http.Handler, error) {
	// If the plugin requires some configuration, it should be under the name of the plugin. E.g.:
	/*
	   "extra_config":{
	       "plugin/http-server":{
	           "name":["krakend-server-example"],
	           "krakend-server-example":{
	               "path": "/some-path"
	           }
	       }
	   }
	*/
	// The config variable contains all the keys you have defined in the configuration
	// if the key doesn't exists or is not a map the plugin returns an error and the default handler
	config, ok := extra[pluginName].(map[string]interface{})
	if !ok {
		return h, errors.New("configuration not found")
	}

	// get config is_show_on_stdout from krakend.json
	isShowOnStdout, _ := config["is_show_on_stdout"].(bool)
	loggers.Debug(fmt.Sprintf("show log on terminal %t", isShowOnStdout))

	// return the actual handler wrapping or your custom logic so it can be used as a replacement for the default http handler
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		pathUrl :=  html.EscapeString(req.URL.Path)

		// If the requested path is what we defined.
		// The path has to be hijacked:
		loggers.Debug("request:", pathUrl)

		logLevel := "SUCCESS"

		reqBody, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(reqBody))

		rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		h.ServeHTTP(rec, req)

		if rec.statusCode >= 400 {
			logLevel = "ERROR"
		}

		logOutput := fmt.Sprintf(LOG_FORMAT, time.Now().Local().Format(LAYOUT_FORMAT), logLevel, req.Header.Get("User-Agent"), req.URL.String(), req.RemoteAddr, req.Method, rec.statusCode, req.Header, req.URL.Query(), string(reqBody), rec.body.String())

		pathName := os.Getenv("LOG_PATH")
		if ok, err := pathExists(pathName); !ok {
			if err != nil {
				fmt.Println(err)
			}

			err := os.Mkdir(pathName, os.ModePerm)
			if err != nil {
				loggers.Error("Failed make directory logs: %v\n", err)
			}
		}

		var (
			file     *os.File
			filename = fmt.Sprintf("%s%s-%s.log", "http-logger", strings.ReplaceAll(pathUrl, "/",  "-"), time.Now().Local().Format("2006-01-02"))
			path     = fmt.Sprintf("%s%s", pathName, filename)
		)

		if _, err := os.Stat(path); err != nil {
			file, err = os.Create(path)
			if err != nil {
				fmt.Printf("Failed create file logs: %v\n", err)
			}
		} else {
			file, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			if err != nil {
				loggers.Error("Failed open file logs: %v\n", err)
			}
		}
		defer file.Close()

		if _, err := file.WriteString(logOutput); err != nil {
			loggers.Error("Could not write to log file:", err)
		}
		loggers.Debug(logOutput)

		rec.flush()

		h.ServeHTTP(w, req)
	}), nil
}

func main() {}

// This logger is replaced by the RegisterLogger method to load the one from KrakenD
var loggers Logger = noopLogger{}

func (registerer) RegisterLogger(v interface{}) {
	l, ok := v.(Logger)
	if !ok {
		return
	}
	loggers = l
	loggers.Debug(fmt.Sprintf("[PLUGIN: %s] Logger loaded", HandlerRegisterer))
}

type Logger interface {
	Debug(v ...interface{})
	Info(v ...interface{})
	Warning(v ...interface{})
	Error(v ...interface{})
	Critical(v ...interface{})
	Fatal(v ...interface{})
}

// Empty logger implementation
type noopLogger struct{}

func (n noopLogger) Debug(_ ...interface{})    {}
func (n noopLogger) Info(_ ...interface{})     {}
func (n noopLogger) Warning(_ ...interface{})  {}
func (n noopLogger) Error(_ ...interface{})    {}
func (n noopLogger) Critical(_ ...interface{}) {}
func (n noopLogger) Fatal(_ ...interface{})    {}


type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.statusCode = code
}

func (rec *responseRecorder) Write(buf []byte) (int, error) {
	rec.body.Write(buf)
	return len(buf), nil
}

func (rec *responseRecorder) flush() {
	rec.ResponseWriter.WriteHeader(rec.statusCode)
	rec.ResponseWriter.Write(rec.body.Bytes())
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
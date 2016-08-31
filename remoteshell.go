//Copyright 2016 myersguo

/**
 * Http Server, serve for execute shell
 * Shell Command Config File: config/shell.go
 * App Config File: conf/app.json
 * Shell Config File: conf/shell.json
 * server for shell name,default get request,return shell shell response  code
 *
 * Next Plan: use mysql store shell job
 */
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

type Command struct {
	Id     int
	Name   string
	Shell  string
	Method string
}

type ServerConfig struct {
	Port   int
	Host   string
	Output bool
}

func GetExecPath() (path string, err error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return
	}
	path, err = filepath.Abs(file)
	if err != nil {
		return
	}
	path = filepath.Dir(path)
	return
}

func loadAppConfig() (config ServerConfig, err error) {
	path, err := GetExecPath()
	if err != nil {
		return
	}
	appconf := path + "/../conf/app.json"
	file, _ := ioutil.ReadFile(appconf)
	if file == nil {
		return
	}
	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Println("json unmarshal failed,err: ", err)
	}
	return
}

func loadShellConfig() (shell []Command, err error) {
	path, err := GetExecPath()
	if err != nil {
		return
	}
	shellconf := path + "/../conf/shell.json"
	file, _ := ioutil.ReadFile(shellconf)
	if file == nil {
		return
	}
	err = json.Unmarshal(file, &shell)
	if err != nil {
		fmt.Println("json unmarshal failed,err: ", err)
	}
	return
}

/**

type httpWriter struct{
    http.ResponseWriter
}

func (w *httpWriter) Response(errno int, errmsg string, data string) {
    w.ResponseWriter.Header().Set("Content-Type", "application/json")
    ret := make(map[string]interface{})
    ret["errno"] = errno
    ret["errmsg"] = errmsg
    ret["data"] = data
    js, err := json.Marshal(ret)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.ResponseWriter.Write(js)
}
*/
func getResponse(errno int, errmsg string, data string) ([]byte, error) {
	ret := make(map[string]interface{})
	ret["errno"] = errno
	ret["errmsg"] = errmsg
	ret["data"] = data
	return json.Marshal(ret)
}

func ExecShellHandler(shell string, config ServerConfig) func(http.ResponseWriter, *http.Request) {
	handler := func(res http.ResponseWriter, req *http.Request) {
		log.Printf("%s %s %s \"%s\"", req.RequestURI, req.RemoteAddr, req.Method, req.UserAgent())
		cmd := exec.Command("/bin/sh", "-c", shell)
		//result, err := cmd.Output()
		result := []byte("Done")
		var err error
		if config.Output {
			result, err = cmd.Output()
		} else {
			err = cmd.Start()
		}
		errcode := 0
		if err != nil {
			log.Println("run shell error: ", err)
			if exiterr, ok := err.(*exec.ExitError); ok {
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					errcode = status.ExitStatus()
				}
			}
			return
		}
		res.Header().Set("Custom-Status", fmt.Sprintf("%d", errcode))
		res.WriteHeader(200)
		out, err := getResponse(200, "ok", string(result))
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Write(out)
		//res.Write([]byte("ok"))

	}
	return handler
}

func setHandlers(commands []Command, config ServerConfig) {
	for _, val := range commands {
		path, shell := val.Name, val.Shell
		log.Println("regester path", path)
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		http.HandleFunc(path, ExecShellHandler(shell, config))
	}
}

func main() {
	config, err := loadAppConfig()
	if err != nil {
		fmt.Println("load app config error,err:", err)
		return
	}
	commands, err := loadShellConfig()
	if err != nil {
		fmt.Println("load app config error,err:", err)
		return
	}
	setHandlers(commands, config)
	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	log.Fatal(http.ListenAndServe(address, nil))
}

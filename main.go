package main

import (
    "encoding/json"
    "fmt"
    "github.com/gorilla/mux"
    "log"
    "net/http"
    "os"
    "os/exec"
    "os/signal"
    "strconv"
    "syscall"
)


type Counter struct {
    cmd         *exec.Cmd
}


func NewCounter() *Counter {
    c := &Counter{}
    c.Initialize()
    return c
}

func (c *Counter) Initialize() {

    c.cmd = nil
}


func (c *Counter) Close() {

    // stop the command if it is running
    if c.cmd != nil {
        c.StopProcess()
    }
}


// start the process
func (c *Counter) StartProcess(startVal int) int {

    // setup the command to run
    c.cmd = exec.Command("./counter.py", "--start", strconv.Itoa(startVal))

    // send cmd's output to this proc's stdout.
    c.cmd.Stdout = os.Stdout

    // set a process group so we can kill the process
    // and all of its children
    c.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

    // start running the command, don't wait for it to complete
    err := c.cmd.Start()
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Process started with PID: %v", c.cmd.Process.Pid)

    return c.cmd.Process.Pid
}


// stop the process
func (c *Counter) StopProcess() {

    // kill the process group of our command
    log.Printf("Killing the process group %v", c.cmd.Process.Pid)
    syscall.Kill(-c.cmd.Process.Pid, syscall.SIGINT)

    // wait for the processes to exit
    log.Printf("Waiting for command to finish...")
    err := c.cmd.Wait()
    log.Printf("Command finished: %v", err)

    // clear out our cmd variable to signal that
    // there is no active command being run.
    c.cmd = nil
}


// remove the logs
func (c *Counter) RemoveLogs() error {

    // setup the command to run
    cmd := exec.Command("rm", "-f", "counter.log")

    // remove the log files
    err := cmd.Run()

    return err
}


// is the process still running?
func (c *Counter) IsRunning() bool {

    return c.cmd != nil

}


type App struct {
    router      *mux.Router
    child       *Counter
    port        int
}


func (a *App) Initialize(port int) {

    a.router = mux.NewRouter()
    a.child = NewCounter()
    a.port = port
}


func (a *App) Close() {

    // unset router
    a.router = nil

    // close the child process
    a.child.Close()
}


func (a *App) Run() {

    a.router.HandleFunc("/cmd/start", a.cmdStartHandler).Methods("POST")
    a.router.HandleFunc("/cmd/stop", a.cmdStopHandler).Methods("GET")
    a.router.HandleFunc("/logs/remove", a.logsRemoveHandler).Methods("GET")

    addr := fmt.Sprintf(":%v",a.port)

    log.Fatal(http.ListenAndServe(addr, a.router))
}


// Handle requests to the /cmd/start API endpoint
func (a *App) cmdStartHandler(w http.ResponseWriter, r *http.Request) {

    type Cmdargs struct {
        Start    int    `json:"start"`
    }

    var cargs Cmdargs
    var err error

    // read the JSON body
    err = json.NewDecoder(r.Body).Decode(&cargs)
    if err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid request payload")
        return
    }
    defer r.Body.Close()

    log.Printf("startVal: %v", cargs.Start)

    pid := a.child.StartProcess(cargs.Start)

    respondWithJSON(w, http.StatusCreated, map[string]int{"pid": pid})
}


// Handle requests to the /cmd/stop API endpoint
func (a *App) cmdStopHandler(w http.ResponseWriter, r *http.Request) {

    // check if cmd was started
    // if not, do nothing
    if ! a.child.IsRunning() {
        log.Printf("Ignoring stop requested before start")
        respondWithError(w, http.StatusBadRequest, "Application not started")
        return
    }

    // stop the process
    a.child.StopProcess()

    respondWithJSON(w, http.StatusOK, map[string]string{"status": "done"})
}


// Handle requests to the /logs/remove API endpoint
func (a *App) logsRemoveHandler(w http.ResponseWriter, r *http.Request) {

    if a.child.IsRunning() {
        log.Printf("Attempt to remove log of running command.")
        log.Printf("Stop command first with /cmd/stop")
        respondWithError(w, http.StatusBadRequest,
            "Cannot clear logs of running command")
        return
    }

    // run the command to remove the logs
    err := a.child.RemoveLogs()
    if err != nil {
        msg := fmt.Sprintf("While removing logs: %v",err)
        respondWithError(w, http.StatusInternalServerError, msg)
        return
    }

    respondWithJSON(w, http.StatusOK, map[string]string{"status": "done"})
}


// helper function from https://bit.ly/2j4nNSs
func respondWithError(w http.ResponseWriter, code int, message string) {
    respondWithJSON(w, code, map[string]string{"error": message})
}


// helper function from https://bit.ly/2j4nNSs
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
    response, _ := json.Marshal(payload)

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(response)
}


// build a router with some api end points
func main() {

    // create our application
    a := &App{}

    a.Initialize(4723)


    // open a channel that we can send signals to
    sigs := make(chan os.Signal, 1)

    // when we get a signal, notify the sigs channel
    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

    // use a goroutine to handle block until we get a signal
    // when a signal arrives on the sigs channel, perform
    // command cleanup
    go func() {

        // block, waiting for a signal
        <-sigs

        // cleanup child processes
        a.Close()

        //exit
        os.Exit(0)
    }()

    // start the server
    a.Run()
}


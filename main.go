package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type JOB struct {
	id    int
	name  string
	jobMS string

	cmd *exec.Cmd

	kill     chan bool
	done     chan bool
	outputCh chan string
}

var valid string
var id int
var jobs map[int]JOB

func getLine(id int, stdout io.ReadCloser) {
	// 创建 scanner 以按行读取输出
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		jobs[id].outputCh <- line
	}

	// 检查是否有错误发生
	if err := scanner.Err(); err != nil {
		if _, exists := jobs[id]; exists {
			log.Println(jobs[id].jobMS+" Error reading output:", err)
		}
	}
}

func waitCMD(id int) {
	// 等待命令完成
	err := jobs[id].cmd.Wait()
	if err != nil {
		if _, exists := jobs[id]; exists {
			log.Println(jobs[id].jobMS+" Error waiting for command to finish:", err)
		}
	} else {
		log.Println(jobs[id].jobMS + " Job Finished")
	}
	jobs[id].done <- true
}

func runCommand(id int) {

	// 执行命令
	cmd := jobs[id].cmd

	// 获取标准输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println(jobs[id].jobMS+" Error creating StdoutPipe:", err)
		close(jobs[id].outputCh)
		return
	}

	// 启动命令
	err = cmd.Start()
	if err != nil {
		log.Println(jobs[id].jobMS+" Error starting command:", err)
		close(jobs[id].outputCh)
		return
	}

	go getLine(id, stdout)

	go waitCMD(id)

	select {
	case <-jobs[id].kill:
		log.Println(jobs[id].jobMS + " Killed by API")
		err := cmd.Process.Kill()
		if err != nil {
			return
		}
	case <-jobs[id].done:
		log.Println(jobs[id].jobMS + " Over")
	}

	// 关闭通道，表示输出结束
	close(jobs[id].outputCh)
}

func runJob(job string) {
	jobMS := "AID:" + strconv.Itoa(id) + " JOB:" + job + " Output"

	// 创建通道来传递输出
	outputCh := make(chan string)
	kill := make(chan bool)
	cmd := exec.Command("sh", "-c", "./"+job+".sh")
	done := make(chan bool)

	jobs[id] = JOB{id: id, name: job, kill: kill, outputCh: outputCh, jobMS: jobMS, cmd: cmd, done: done}

	// 启动命令执行协程
	go runCommand(id)

	// 在主协程中接收并打印输出
	for output := range outputCh {
		log.Println(jobMS, output)
	}

	delete(jobs, id)
}

func getToken() {
	// 读取文件内容
	content, err := os.ReadFile("token")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// 将文件内容转换为字符串
	valid = strings.TrimSpace(string(content))
}

func JobHandler(w http.ResponseWriter, r *http.Request) {
	// 获取查询参数
	job := r.URL.Query().Get("job")
	token := r.URL.Query().Get("token")

	// 检查是否存在 job 参数
	if job == "" {
		http.Error(w, "Missing 'job' parameter", http.StatusBadRequest)
		return
	}

	// 检查 token 合法
	if token != valid {
		http.Error(w, "Token wrong", http.StatusBadRequest)
		log.Println("job " + job + "`s token invalid")
		return
	}

	log.Println("job " + job + " start")

	jobName := "./" + job + ".sh"
	_, err := os.Stat(jobName)
	// 输出响应
	if err != nil {
		fmt.Fprintf(w, "Can not find job")
		log.Println("Can not find job" + job)
		return
	}

	id++
	go runJob(job)

	fmt.Fprintf(w, "job %s start, id is %d", job, id)
}

func killHandler(w http.ResponseWriter, r *http.Request) {
	// 获取查询参数
	kid := r.URL.Query().Get("id")
	token := r.URL.Query().Get("token")

	// 检查是否存在 job 参数
	if kid == "" {
		http.Error(w, "Missing 'id' parameter", http.StatusBadRequest)
		return
	}

	// 检查 token 合法
	if token != valid {
		http.Error(w, "Token wrong", http.StatusBadRequest)
		log.Println("job`s token invalid")
		return
	}

	kil, err := strconv.Atoi(kid)
	if err != nil {
		fmt.Fprintf(w, "job id`s format is wrong")
		return
	}

	if jobs[kil].id == 0 {
		fmt.Fprintf(w, "job not exists")
		return
	}

	jobs[kil].kill <- true

	fmt.Fprintf(w, "job killed")
}

func main() {
	getToken()
	if valid == "" {
		log.Println("Service failed")
		return
	}
	jobs = make(map[int]JOB)
	id = 0

	log.Println("token is " + valid)

	http.HandleFunc("/job", JobHandler)
	http.HandleFunc("/kill", killHandler)
	log.Println("Server is listening on :9922...")
	http.ListenAndServe("0.0.0.0:9922", nil)
}

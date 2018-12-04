package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"szcheck/lib/logger"
	"time"
)

type ModeList struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type PageContent struct {
	Day      int    `json:"day"`
	End      string `json:"end"`
	FromCity string `json:"fromCity"`
	Id       int    `json:"id"`
	ImgUrl   string `json:"imgUrl"`
	Mode     string `json:"mode"`
	ModeId   int    `json:"modeId"`
	Price    int    `json:"price"`
	Start    string `json:"start"`
	ToCity   string `json:"toCity"`
}

type Page struct {
	Contents   []PageContent `json:"content"`
	FirstPage  bool          `json:"firstPage"`
	LastPage   bool          `json:"lastPage"`
	Number     int           `json:"number"`
	Size       int           `json:"size"`
	Total      int           `json:"total"`
	TotalPages int           `json:"totalPages"`
}

type Carsslice struct {
	ModeLists  []ModeList `json:"modeList"`
	StartDate  string     `json:"startDate"`
	EndDate    string     `json:"endDate"`
	FromCityId string     `json:"fromCityId"`
	Pages      Page       `json:"page"`
	ToCity     string     `json:"toCity"`
	ToCityId   string     `json:"toCityId"`
	FromCity   string     `json:"fromCity"`
}

func main() {
	hasCheckOut := false
	checkCount := 1

	var cars Carsslice
	url := "https://easyride.zuche.com/member/windCarList.do?callback=jQuery17107594033539615033_1543829205606&fromCity=%25E6%25B7%25B1%25E5%259C%25B3&fromCityId=15&toCity=%25E8%25AF%25B7%25E9%2580%2589%25E6%258B%25A9%25E8%25BF%2598%25E8%25BD%25A6%25E5%259F%258E%25E5%25B8%2582&toCityId=&sortBy=price&sort=0&_=1543829311661"
	for range time.Tick(10 * time.Second) {
		if !hasCheckOut {
			logger.Notice("查询租车信息，当前第", checkCount, "次查询")
			checkCount++
			resp, err := http.Get(url)
			if err != nil {
				logger.Notice("http.Get error")
			}
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logger.Notice("ioutil.ReadAll error")
			}
			frist := strings.Split(string(body), "(")[1]
			carStr := strings.Split(string(frist), ")")[0]
			err = json.Unmarshal([]byte(carStr), &cars)
			for _, content := range cars.Pages.Contents {
				if content.ToCity == "武汉" {
					logger.Notice("找到租车信息")
					sendEmailTls()
					hasCheckOut = true
				}
			}
			if !hasCheckOut {
				logger.Notice("没有找到对应的租车信息")
			}
		} else {
			logger.Notice("已找到租车信息")
		}
	}
}

//普通25端口发送
func sendEmail() {
	UserEmail := "yg_yigeng@163.com"
	Mail_Smtp_Port := ":25"
	Mail_Password := "YIgeng520"
	Mail_Smtp_Host := "smtp.163.com"
	auth := smtp.PlainAuth("", UserEmail, Mail_Password, Mail_Smtp_Host)
	to := []string{"yg_yigeng@163.com"}
	nickname := "self"
	user := UserEmail

	subject := "ShenZhou Check Success"
	content_type := "Content-Type:text/html;charset=utf-8"
	body := "已找到神州到武汉的顺风车，快去app下单！"
	msg := []byte("To: " + strings.Join(to, ",") + "\r\nFrom: " + nickname +
		"<" + user + ">\r\nSubject: " + subject + "\r\n" + content_type + "\r\n\r\n" + body)
	err := smtp.SendMail(Mail_Smtp_Host+Mail_Smtp_Port, auth, user, to, msg)
	if err != nil {
		fmt.Printf("send mail error: %v", err)
	} else {
		fmt.Printf("邮件发送成功\n")
	}
}

//加密发送
func sendEmailTls() {
	UserEmail := "yg_yigeng@163.com"
	Mail_Smtp_Port := ":465"
	Mail_Password := "YIgeng520"
	Mail_Smtp_Host := "smtp.163.com"
	to := "yg_yigeng@163.com"

	header := make(map[string]string)

	header["From"] = "test" + "<" + UserEmail + ">"
	header["To"] = to
	header["Subject"] = "ShenZhou Check Success"
	header["Content-Type"] = "text/html;chartset=UTF-8"
	body := "已找到神州到武汉的顺风车，快去app下单！ "
	message := ""

	for k, v := range header {
		message += fmt.Sprintf("%s:%s\r\n", k, v)
	}

	message += "\r\n" + body

	auth := smtp.PlainAuth(
		"",
		UserEmail,
		Mail_Password,
		Mail_Smtp_Host,
	)

	err := SendMailUsingTLS(
		Mail_Smtp_Host+Mail_Smtp_Port,
		auth,
		UserEmail,
		[]string{to},
		[]byte(message),
	)

	if err != nil {
		panic(err)
	}
}

//return a smtp client
func Dial(addr string) (*smtp.Client, error) {
	conn, err := tls.Dial("tcp", addr, nil)
	if err != nil {
		log.Panicln("Dialing Error:", err)
		return nil, err
	}
	//分解主机端口字符串
	host, _, _ := net.SplitHostPort(addr)
	return smtp.NewClient(conn, host)
}

//参考net/smtp的func SendMail()
//使用net.Dial连接tls(ssl)端口时,smtp.NewClient()会卡住且不提示err
//len(to)>1时,to[1]开始提示是密送
func SendMailUsingTLS(addr string, auth smtp.Auth, from string,
	to []string, msg []byte) (err error) {

	//create smtp client
	c, err := Dial(addr)
	if err != nil {
		log.Println("Create smpt client error:", err)
		return err
	}
	defer c.Close()

	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err = c.Auth(auth); err != nil {
				log.Println("Error during AUTH", err)
				return err
			}
		}
	}

	if err = c.Mail(from); err != nil {
		return err
	}

	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}

	w, err := c.Data()
	if err != nil {
		return err
	}

	_, err = w.Write(msg)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}

package controllers

import (
	"github.com/astaxie/beego"
	"crypto/md5"
	"io"
	"encoding/hex"
	"path"
	"github.com/qiniu/api.v7/storage"
	"fmt"
	"github.com/qiniu/api.v7/auth/qbox"
	"os"
	"context"
)

// Operations about Users
type UploadController struct {
	beego.Controller
}

func (this *UploadController) Post() {
	file, h, err := this.GetFile("file")

	if err != nil {
		this.Error(1,"没有上传内容")
	}

	//计算文件md5，使同一文件在七牛上只有一份
	md5 := md5.New()
	io.Copy(md5,file)
	md5Str := hex.EncodeToString(md5.Sum(nil))

	defer file.Close()

	localFile := h.Filename

	//存入本地文件
	this.SaveToFile("file",localFile)
	key := md5Str+path.Ext(localFile)

	type QiniuConfig struct {
		Ak string
		Sk string
		Bucket string
		Domain string
	}
	config := QiniuConfig{
		Ak:beego.AppConfig.String("qiniu.ak"),
		Sk:beego.AppConfig.String("qiniu.sk"),
		Bucket:beego.AppConfig.String("qiniu.bucket"),
		Domain:beego.AppConfig.String("qiniu.domain"),
	}

	if config.Ak == "" || config.Sk == "" || config.Bucket == "" || config.Domain == ""{

		this.Error(1,"服务器没有设置七牛云，请联系管理员")
	}

	putPolicy := storage.PutPolicy{
		Scope: config.Bucket,
	}
	mac := qbox.NewMac(config.Ak, config.Sk)

	upToken := putPolicy.UploadToken(mac)
	cfg := storage.Config{}
	// 空间对应的机房
	cfg.Zone = &storage.ZoneHuanan
	// 是否使用https域名
	cfg.UseHTTPS = false
	// 上传是否使用CDN上传加速
	cfg.UseCdnDomains = false
	// 构建表单上传的对象
	formUploader := storage.NewFormUploader(&cfg)
	ret := storage.PutRet{}
	// 可选配置
	putExtra := storage.PutExtra{
		Params: map[string]string{},
	}
	err = formUploader.PutFile(context.Background(), &ret, upToken, key, localFile, &putExtra)
	if err != nil {
		fmt.Println(err)
		return
	}

	//删除本地文件
	os.Remove(localFile)

	this.Success(map[string]interface{}{"src":"http://"+config.Domain+"/"+ret.Key,"name":localFile},"success")
}

func (this *UploadController) Success(data interface{}, msg string) {

	this.Data["json"] = Response{0,msg,data}
	this.ServeJSON()
	this.StopRun()
}
func (this *UploadController) Error(code int, Msg string) {

	if code == 0 {
		if Msg == "" {
			Msg = "success"
		}
		this.Data["json"] = Response{code,Msg,nil}
	} else {
		this.Data["json"] = Response{code,Msg,nil}
	}

	this.ServeJSON()
	this.StopRun()
}
// Controller Response is controller error info struct.
type Response struct {
	Code int `json:"code"`
	Msg string `json:"msg"`
	Data interface{} `json:"data"`
}
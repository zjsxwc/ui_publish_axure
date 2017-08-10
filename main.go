package main

import (
	"gopkg.in/baa.v1"
	"os"
	"fmt"
	"strings"
	"errors"
	"io"
	"archive/zip"
	"path/filepath"
	"time"
)


func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}


func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func substr(str string, start int, end int) string {
	rs := []rune(str)
	length := len(rs)

	if start < 0 || start > length {
		return ""
	}

	if end < 0 || end > length {
		return ""
	}
	return string(rs[start:end])
}

func main() {

	dir, _ := os.Getwd()

	fmt.Println(strings.Replace(dir, " ", "\\ ", -1))
	fmt.Println(time.Now().Unix())

	app := baa.New()

	app.Static("/assets", dir + "/assets", true, func(c *baa.Context) {
		// 你可以对输出的结果干点啥的
	})


	app.Get("/", func(c *baa.Context) {
		indexHtml := `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>upload zip file</title>
</head>
<body>

  <form action="/upload-zip-file" method="post" enctype="multipart/form-data" target="_blank">
      <input type="file" name="zipFile">
      <button type="submit">upload</button>
  </form>

</body>
</html>
		`

		c.Text(200, []byte(indexHtml))
	})



	app.Post("/upload-zip-file", func (c *baa.Context) {
		file, header, err := c.GetFile("zipFile")
		if err != nil {
			c.Error(errors.New("没有文件被上传"))
			return
		}
		defer file.Close()

		savedTo := "package.zip"
		newFile, err := os.Create(savedTo)
		if err != nil {
			c.Error(errors.New("文件创建失败"))
			return
		}
		defer newFile.Close()

		size, err := io.Copy(newFile, file)

		relativePathRoot := "/assets/"+ fmt.Sprintf("%d", time.Now().Unix())
		extractDir := dir + relativePathRoot
		RemoveContents(extractDir)
		Unzip(savedTo,  extractDir)

		msg := ""
		msg = fmt.Sprintf("fileName: %s, savedTo: %s, size: %d, err: %v", header.Filename, savedTo, size, err)
		fmt.Println(msg)


		contentTmpl := `
			<h2>上传成功</h2>
			<br>
			<h3>本次上传根路径是 %s</h3>
			<a href="%s" target="_blank">尝试访问index.html</a>
			<br>
			<h3>%s</h3>

		`
		msg = fmt.Sprintf(contentTmpl, relativePathRoot, relativePathRoot + "/index.html" , msg)

		c.Text(200, []byte(msg))
	})
	app.Run(":1323")
}

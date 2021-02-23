package netrc

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
)

// GenerateDotNetrcFile 生成 .netrc 文件
func GenerateDotNetrcFile() error {
	githubTokenLogin := os.Getenv("GITHUB_TOKEN_LOGIN")
	githubTokenPassword := os.Getenv("GITHUB_TOKEN_PASSWORD")
	if githubTokenLogin == "" || githubTokenPassword == "" {
		return errors.New("The env GITHUB_TOKEN_LOGIN or GITHUB_TOKEN_PASSWORD cannot be blank")
	}
	bashScript := `
#!/bin/sh -e
cat << EOF > /root/.netrc
machine github.com
login ${GITHUB_TOKEN_LOGIN}
password ${GITHUB_TOKEN_PASSWORD}
EOF
`
	command := os.ExpandEnv(bashScript)
	cmd := exec.Command("/bin/sh", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return errors.New(stderr.String())
	}
	return nil
}

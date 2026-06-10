package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/GMWalletApp/epusdt/command"
	"github.com/GMWalletApp/epusdt/config"
	"github.com/gookit/color"
)

//go:embed all:www
var wwwDir embed.FS

func executableDir() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", err
	}
	return filepath.Dir(exePath), nil
}

func extractEmbeddedDir(src embed.FS, embeddedRoot, dstRoot string) error {
	sub, err := fs.Sub(src, embeddedRoot)
	if err != nil {
		return err
	}

	return fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dstRoot, path)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}

		in, err := sub.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, in)
		return err
	})
}

func releaseStatic(src embed.FS, target string) (string, error) {
	baseDir, err := executableDir()
	if err != nil {
		return "", err
	}

	targetDir := filepath.Join(baseDir, target)
	if err := os.RemoveAll(targetDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}
	if err := extractEmbeddedDir(src, target, targetDir); err != nil {
		return "", err
	}
	return targetDir, nil
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			color.Error.Println("[Start Server Err!!!] ", err)
		}
	}()
	command.RegisterStaticReleaser(func() error {
		if !config.GatewayUIEnabled() {
			return nil
		}
		wwwPath, err := releaseStatic(wwwDir, "www")
		if err != nil {
			return err
		}
		fmt.Println("www released to:", wwwPath)
		return nil
	})
	if err := command.Execute(); err != nil {
		panic(err)
	}
}

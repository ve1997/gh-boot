package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/cli/go-gh"
	"github.com/eiannone/keyboard"
)

func main() {
	bootApp()
}

func bootApp() {
	// 1. プロジェクト名の入力
	projectName, err := promptWithPlaceholder("What is your project named?", "my-app")
	if err != nil {
		fmt.Println("Error reading project name:", err)
		os.Exit(1)
	}

	// 2. フレームワークの選択（矢印キーで選択）
	templates := []string{"nextjs@latest", "nextjs@canary", "react", "cdk"}
	var template string
	templatePrompt := &survey.Select{
		Message: "Which template do you want to use? ... ",
		Options: templates,
	}
	err = survey.AskOne(templatePrompt, &template)
	if err != nil {
		handleSurveyError(err)
	}

	// 3. パッケージマネージャーの選択（矢印キーで選択）
	packageManagers := []string{"pnpm", "bun", "npm", "yarn"}
	var packageManager string
	packageManagerPrompt := &survey.Select{
		Message: "Which package-manager do you want to use? ... ",
		Options: packageManagers,
	}
	err = survey.AskOne(packageManagerPrompt, &packageManager)
	if err != nil {
		handleSurveyError(err)
	}

	// 4. フレームワークごとの処理
	switch template {
	case "nextjs@latest":
		fmt.Printf("\n🚀 \x1b[90m> \x1b[1mnpx create-next-app@latest %s --use-%s --ts --app --tailwind --no-eslint --no-src-dir --no-import-alias\x1b[0m\n\n", projectName, packageManager)
		runCommand("npx", "create-next-app@latest", projectName, "--use-"+packageManager, "--ts", "--app", "--tailwind", "--no-eslint", "--no-src-dir", "--no-import-alias")
	case "nextjs@canary":
		fmt.Printf("\n🚀 \x1b[90m> \x1b[1mnpx create-next-app@canary %s --use-%s --ts --app --tailwind --no-eslint --no-src-dir --no-import-alias --turbo\x1b[0m\n\n", projectName, packageManager)
		runCommand("npx", "create-next-app@canary", projectName, "--use-"+packageManager, "--ts", "--app", "--tailwind", "--no-eslint", "--no-src-dir", "--no-import-alias", "--turbo")
	case "react":
		fmt.Printf("\n🚀 \x1b[90m> \x1b[1m%s create vite@latest %s --template react-ts\x1b[0m\n", packageManager, projectName)
		runCommand(packageManager, "create", "vite@latest", projectName, "--template", "react-ts")
	case "cdk":
		fmt.Printf("\n🚀 \x1b[90m> \x1b[1mcdk init app -l typescript\x1b[0m\n\n")
		os.Mkdir(projectName, 0755)
		os.Chdir(projectName)
		runCommand("cdk", "init", "app", "-l", "typescript")

		var installCmd string
		if packageManager == "npm" {
			installCmd = "i"
		} else {
			installCmd = "add"
		}

		// package-lock.jsonを削除して指定したパッケージマネージャーで再インストール
		os.Remove("package-lock.json")
		runCommand(packageManager, "install")
		runCommand(packageManager, installCmd, "hono")
		os.Mkdir("lambda", 0755)
		file, err := os.Create("lambda/index.ts")
		if err != nil {
			fmt.Println("Error creating lambda/index.ts:", err)
			os.Exit(1)
		}
		file.Close()
	default:
		fmt.Println("Invalid framework selection.")
		os.Exit(1)
	}

	// 5. pushするための共通処理
	if template != "cdk" {
		err := os.Chdir(projectName)
		if err != nil {
			fmt.Printf("Failed to change directory to %s\n", projectName)
			os.Exit(1)
		}
		if template == "react" {
			runCommand(packageManager, "install")
			runCommand("git", "init")
			runCommand("git", "add", ".")
			runCommand("git", "commit", "-m", "Initial commit from Create React App via Vite")
		}
	}
	fmt.Printf("\n🚀 \x1b[1mLocal project created!\x1b[0m 🚀\n\n")

	// 6. リポジトリの可視性の選択（矢印キーで選択）
	repoVisibilities := []string{"public", "private", "internal", "none"}
	var repoVisibility string
	repoVisibilityPrompt := &survey.Select{
		Message: "What visibility would you like for your remote repository? ... ",
		Options: repoVisibilities,
	}
	err = survey.AskOne(repoVisibilityPrompt, &repoVisibility)
	if err != nil {
		handleSurveyError(err)
	}

	// 7. リモートリポジトリの作成
	if repoVisibility != "none" {
		runGhCommand("repo", "create", projectName, "--source", ".", "--"+repoVisibility, "--remote", "gh-origin", "--push")
		fmt.Printf("\n🚀 \x1b[1mRemote repository created and main branch pushed!\x1b[0m 🚀\n")
	}

	// 8. finish
	os.Chdir("..")
	fmt.Printf("\n🚀 \x1b[1mAll done!\x1b[0m 🚀\n")
	fmt.Printf("\x1b[90mnext: \x1b[1mcode %s/\x1b[0m\n", projectName)
}

// プレースホルダー対応の入力関数
func promptWithPlaceholder(promptMessage string, placeholder string) (string, error) {
	// 初期プロンプトの表示
	fmt.Printf("\x1b[32m?\x1b[0m \x1b[1m%s\x1b[0m ... \x1b[90m(%s)\x1b[0m", promptMessage, placeholder)
	input := ""
	placeholderVisible := true

	// キーボード入力を開始
	err := keyboard.Open()
	if err != nil {
		return "", err
	}
	defer keyboard.Close()

	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			return "", err
		}

		switch key {
		case keyboard.KeyEnter:
			if input == "" && placeholderVisible {
				input = placeholder
			}
			// カーソルを行頭に戻し、行をクリアして入力値を水色で表示
			fmt.Printf("\r\x1b[K\x1b[32m?\x1b[0m \x1b[1m%s\x1b[0m ... \x1b[36m%s\x1b[0m\n", promptMessage, input)
			// 入力が空でないことを確認
			if input == "" {
				fmt.Println("Error: Input cannot be empty.")
				fmt.Printf("\x1b[32m?\x1b[0m \x1b[1m%s\x1b[0m ... \x1b[90m(%s)\x1b[0m", promptMessage, placeholder)
				input = ""
				placeholderVisible = true
				continue
			}
			return input, nil

		case keyboard.KeyBackspace, keyboard.KeyBackspace2:
			if len(input) > 0 {
				input = input[:len(input)-1]
				// バックスペースで文字を削除
				fmt.Print("\b \b")
			}

		case keyboard.KeyCtrlC, keyboard.KeyCtrlD:
			fmt.Println("\nOperation cancelled.")
			os.Exit(1)

		default:
			if placeholderVisible {
				// プレースホルダーを消去
				fmt.Printf("\r\x1b[K\x1b[32m?\x1b[0m \x1b[1m%s\x1b[0m ... ", promptMessage)
				placeholderVisible = false
			}
			input += string(char)
			fmt.Printf("%s", string(char))
		}

		// 入力が空でなくなった場合にプレースホルダーを再表示
		if input == "" && !placeholderVisible {
			fmt.Printf("\x1b[90m(%s)\x1b[0m", placeholder)
			placeholderVisible = true
		}
	}
}

// コマンドを実行する関数
func runCommand(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Failed to run command %s %v: %v\n", name, args, err)
		os.Exit(1)
	}
}

// ghコマンドを実行する関数
func runGhCommand(args ...string) {
	stdOut, stdErr, err := gh.Exec(args...)
	if err != nil {
		fmt.Printf("Error running gh command: %v\n", err)
		fmt.Println(stdErr.String())
		os.Exit(1)
	}
	fmt.Println(stdOut.String())
}

// survey.AskOne のエラーハンドリング関数
func handleSurveyError(err error) {
	if err == nil {
		return
	}
	// エラーメッセージに "interrupt" が含まれている場合は Ctrl+C とみなす
	if strings.Contains(strings.ToLower(err.Error()), "interrupt") {
		fmt.Println("\nOperation cancelled.")
		os.Exit(1)
	}
	// その他のエラー
	fmt.Println("Error:", err)
	os.Exit(1)
}

package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// downloadFile downloads a file from URL to local path
func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// downloadNgrok downloads ngrok for the current platform
func downloadNgrok() error {
	var downloadURL string
	var filename string

	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH == "amd64" {
			downloadURL = "https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-windows-amd64.zip"
			filename = "ngrok.zip"
		} else {
			downloadURL = "https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-windows-386.zip"
			filename = "ngrok.zip"
		}
	case "linux":
		if runtime.GOARCH == "amd64" {
			downloadURL = "https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-linux-amd64.tgz"
			filename = "ngrok.tgz"
		} else if runtime.GOARCH == "arm64" {
			downloadURL = "https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-linux-arm64.tgz"
			filename = "ngrok.tgz"
		} else {
			downloadURL = "https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-linux-386.tgz"
			filename = "ngrok.tgz"
		}
	case "darwin":
		if runtime.GOARCH == "amd64" {
			downloadURL = "https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-darwin-amd64.zip"
			filename = "ngrok.zip"
		} else {
			downloadURL = "https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-darwin-arm64.zip"
			filename = "ngrok.zip"
		}
	default:
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("📥 Downloading ngrok from %s...\n", downloadURL)
	err := downloadFile(downloadURL, filename)
	if err != nil {
		return fmt.Errorf("failed to download ngrok: %v", err)
	}

	// Extract the archive
	fmt.Println("📦 Extracting ngrok...")
	if strings.HasSuffix(filename, ".zip") {
		cmd := exec.Command("unzip", "-o", filename)
		if runtime.GOOS == "windows" {
			cmd = exec.Command("powershell", "Expand-Archive", "-Path", filename, "-DestinationPath", ".", "-Force")
		}
		err = cmd.Run()
	} else if strings.HasSuffix(filename, ".tgz") {
		cmd := exec.Command("tar", "-xzf", filename)
		err = cmd.Run()
	}

	if err != nil {
		return fmt.Errorf("failed to extract ngrok: %v", err)
	}

	// Clean up archive
	os.Remove(filename)

	// Make ngrok executable on Unix systems
	if runtime.GOOS != "windows" {
		os.Chmod("ngrok", 0755)
	}

	fmt.Println("✅ ngrok downloaded and extracted successfully!")
	return nil
}

// checkDockerRunning checks if Docker is running
func checkDockerRunning() error {
	cmd := exec.Command("docker", "version")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Docker is not running or not installed. Please start Docker Desktop.")
	}
	return nil
}

// startMinecraftContainer starts the Minecraft Docker container
func startMinecraftContainer() (*exec.Cmd, error) {
	fmt.Println("🐳 Starting Minecraft Docker container...")

	// Stop any existing container first
	exec.Command("docker", "stop", "minecraft-server").Run()
	exec.Command("docker", "rm", "minecraft-server").Run()

	// Start the container
	cmd := exec.Command("docker", "run",
		"--name", "minecraft-server",
		"-p", "25565:25565",
		"-d",
		"--restart", "unless-stopped",
		"superwang0603/minecraft-server:latest")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to start Minecraft container: %v\nOutput: %s", err, string(output))
	}

	fmt.Println("✅ Minecraft container started successfully!")
	fmt.Println("🎮 Server is running on localhost:25565")

	return cmd, nil
}

// killExistingNgrok kills any existing ngrok processes
func killExistingNgrok() {
	fmt.Println("🔄 Checking for existing ngrok sessions...")

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("taskkill", "/f", "/im", "ngrok.exe")
	} else {
		cmd = exec.Command("pkill", "ngrok")
	}

	err := cmd.Run()
	if err == nil {
		fmt.Println("✅ Stopped existing ngrok session")
		time.Sleep(2 * time.Second) // Wait for cleanup
	} else {
		fmt.Println("ℹ️  No existing ngrok sessions found")
	}
}

// clearNgrokConfig clears any problematic ngrok configuration
func clearNgrokConfig() {
	fmt.Println("🧹 Clearing ngrok configuration...")

	// Remove any existing config file that might cause issues
	configPath := ""
	if runtime.GOOS == "windows" {
		configPath = filepath.Join(os.Getenv("APPDATA"), "ngrok", "ngrok.yml")
	} else {
		configPath = filepath.Join(os.Getenv("HOME"), ".config", "ngrok", "ngrok.yml")
	}

	if _, err := os.Stat(configPath); err == nil {
		os.Remove(configPath)
		fmt.Println("✅ Cleared ngrok config file")
	}
}

// startNgrok starts ngrok tunnel for the Minecraft port
func startNgrok(ngrokToken string) (*exec.Cmd, error) {
	// Kill any existing ngrok sessions first
	killExistingNgrok()

	// Clear any problematic configuration
	clearNgrokConfig()

	// Check if ngrok exists in PATH first
	ngrokPath := "ngrok"
	if runtime.GOOS == "windows" {
		ngrokPath = "ngrok.exe"
	}

	// First try to find ngrok in PATH
	_, err := exec.LookPath(ngrokPath)
	if err != nil {
		// If not in PATH, check current directory
		if _, err := os.Stat(ngrokPath); os.IsNotExist(err) {
			fmt.Println("ngrok not found in PATH or current directory. Downloading...")
			if err := downloadNgrok(); err != nil {
				return nil, fmt.Errorf("failed to download ngrok: %v", err)
			}
		}
	} else {
		fmt.Println("✅ Found ngrok in system PATH")
	}

	// Authenticate ngrok if token provided
	if ngrokToken != "" {
		fmt.Println("🔐 Authenticating ngrok...")
		authCmd := exec.Command(ngrokPath, "config", "add-authtoken", ngrokToken)
		if err := authCmd.Run(); err != nil {
			fmt.Printf("⚠️  Warning: Failed to authenticate ngrok: %v\n", err)
		} else {
			fmt.Println("✅ ngrok authenticated!")
		}
	}

	// Start ngrok tunnel for port 25565
	fmt.Println("🌐 Starting ngrok tunnel for Minecraft port 25565...")
	fmt.Printf("📋 Running command: %s tcp 25565\n", ngrokPath)
	cmd := exec.Command(ngrokPath, "tcp", "25565", "--log=stdout")

	// Set up output pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ngrok: %v", err)
	}

	// Start goroutines to handle output
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf("📤 [NGROK OUTPUT] %s\n", line)

			// Look for the public URL in multiple formats
			if strings.Contains(line, "tcp://") || strings.Contains(line, "ngrok.io") {
				parts := strings.Fields(line)
				for _, part := range parts {
					if strings.HasPrefix(part, "tcp://") {
						fmt.Printf("\n🎮🎮🎮 MINECRAFT SERVER READY! 🎮🎮🎮\n")
						fmt.Printf("🌍 Public Address: %s\n", part)
						fmt.Printf("📝 Share this address with your friends!\n")
						fmt.Printf("🔗 They can connect using: %s\n", part)
						fmt.Printf("🌐 Or check ngrok dashboard: http://localhost:4040\n")
						fmt.Printf("🎮🎮🎮🎮🎮🎮🎮🎮🎮🎮🎮🎮🎮🎮🎮🎮🎮🎮\n\n")
						break
					}
				}
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf("⚠️  [NGROK ERROR] %s\n", line)
		}
	}()

	// Wait a moment for ngrok to start
	time.Sleep(3 * time.Second)

	fmt.Println("✅ ngrok tunnel started!")
	return cmd, nil
}

// showContainerLogs shows the Minecraft container logs
func showContainerLogs() {
	fmt.Println("📋 Minecraft Server Logs:")
	fmt.Println("=========================")

	cmd := exec.Command("docker", "logs", "-f", "minecraft-server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		cmd.Run()
	}()
}

func main() {
	fmt.Println("🎮 Minecraft Docker + Ngrok Deployment Tool")
	fmt.Println("===========================================")
	fmt.Println("This will:")
	fmt.Println("1. Start your Minecraft Docker container on port 25565")
	fmt.Println("2. Create an ngrok tunnel to expose it to the internet")
	fmt.Println("3. Give you a public address to share with friends")
	fmt.Println()

	// Check if Docker is running
	if err := checkDockerRunning(); err != nil {
		log.Fatalf("❌ %v", err)
	}

	// Get ngrok token from user
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your ngrok authtoken (optional, press Enter to skip): ")
	ngrokToken, _ := reader.ReadString('\n')
	ngrokToken = strings.TrimSpace(ngrokToken)

	// Start Minecraft container
	mcCmd, err := startMinecraftContainer()
	if err != nil {
		log.Fatalf("❌ Failed to start Minecraft container: %v", err)
	}

	// Wait a moment for container to fully start
	fmt.Println("⏳ Waiting for Minecraft server to start...")
	time.Sleep(10 * time.Second)

	// Start ngrok tunnel
	ngrokCmd, err := startNgrok(ngrokToken)
	if err != nil {
		log.Fatalf("❌ Failed to start ngrok: %v", err)
	}

	// Show container logs in background
	go showContainerLogs()

	fmt.Println("\n🎉 Deployment complete!")
	fmt.Println("Your Minecraft server is now running and accessible via ngrok!")
	fmt.Println("\n📋 Commands:")
	fmt.Println("- View logs: docker logs -f minecraft-server")
	fmt.Println("- Stop server: docker stop minecraft-server")
	fmt.Println("- Restart server: docker restart minecraft-server")
	fmt.Println("\nPress Ctrl+C to stop both the server and ngrok tunnel...")

	// Wait for interrupt signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Wait for both processes
	done := make(chan error, 2)

	go func() {
		done <- mcCmd.Wait()
	}()

	go func() {
		done <- ngrokCmd.Wait()
	}()

	// Wait for either process to finish or interrupt
	select {
	case err := <-done:
		if err != nil {
			fmt.Printf("Process finished with error: %v\n", err)
		}
	case <-ctx.Done():
		fmt.Println("\n🛑 Shutting down...")
		exec.Command("docker", "stop", "minecraft-server").Run()
		ngrokCmd.Process.Kill()
	}

	fmt.Println("👋 Goodbye!")
}

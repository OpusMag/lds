# lds

An alternative to the `ls` command that makes it easier to navigate complicated directories with lots of files.

![lds](https://github.com/user-attachments/assets/f38c2770-5e40-4886-b6b0-4e444f298942)

## Table of Contents

- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
  - [Linux](#linux)
  - [Windows](#windows)
  - [macOS](#macos)
- [Getting Started](#getting-started)
- [Usage](#usage)
- [Configuration](#configuration)
- [Key Bindings](#key-bindings)
- [Contributing](#contributing)
- [Support](#support)
- [License](#license)

## Features

- Easy navigation in complex directories
- Search functionality
- File and directory operations (rename, move, delete, copy)
- Git repository status
- Customizable key bindings and themes

## Prerequisites

- Git
- Go programming language

## Installation

### Linux

#### Dependencies

Install Git and Go:

- Arch Linux:
  ```sh
  sudo pacman -S git go
  ```
- Fedora:
  ```sh
  sudo dnf install git-all go
  ```
- Ubuntu/Debian:
  ```sh
  sudo apt install git-all golang-go
  ```
- Snap:
  ```sh
  sudo snap install --classic go
  ```

#### Install `lds`

Option A: Download the binary

1. Go to the [latest release](https://github.com/OpusMag/lds) and download the binary.
2. Make it executable:
   ```sh
   chmod +x ~/Downloads/lds
   ```
3. Move it to `/usr/local/bin/`:
   ```sh
   sudo mv ~/Downloads/lds /usr/local/bin/
   ```

Option B: Build from source

1. Clone the repository:
   ```sh
   git clone https://github.com/OpusMag/lds
   cd lds
   ```
2. Build the binary:
   ```sh
   go mod tidy
   go build -o lds main.go
   ```
3. Move it to `/usr/local/bin/`:
   ```sh
   sudo mv lds /usr/local/bin/
   ```

### Windows

#### Dependencies

Install Git and Go:

- Git: [Download Git](https://git-scm.com/download/win) and install it.
- Go: [Download Go](https://golang.org/dl/) and follow the [installation guide](https://www.geeksforgeeks.org/how-to-install-go-on-windows/).

#### Install `lds`

Option A: Download the binary

1. Go to the [latest release](https://github.com/OpusMag/lds) and download `lds.exe`.
2. Move the binary to a directory (e.g., `C:\Tools`).
3. Add the directory to PATH:
   - Press `Win + X` and select `System`.
   - Click on `Advanced system settings`.
   - In the `System properties` window, click on `Environment variables`.
   - Find the `Path` variable in the `System variables` section and click `Edit`.
   - Click `New` and add the path to the directory where you placed `lds.exe` (e.g., `C:\Tools`).

Option B: Build from source

1. Clone the repository:
   ```sh
   git clone https://github.com/OpusMag/lds
   cd lds
   ```
2. Build the binary:
   ```sh
   go mod tidy
   go build -o lds.exe main.go
   ```

### Create a directory for config and copy the config to it

You need to create a directory for the config file and copy the config file to it (or make your own).

```mkdir "C:\Users\YOURUSERNAME\AppData\Local\lds"
copy "C:\Users\YOURUSERNAME\Downloads\lds\config.json" "C:\Users\YOURUSERNAME\AppData\Local\lds\config.json"```

### macOS

#### Dependencies

Install Git and Go:

- Git:
  - Using Homebrew (Recommended):
    ```sh
    brew install git
    ```
  - Using Xcode Command Line Tools:
    ```sh
    xcode-select --install
    ```
- Go:
  - Using Homebrew (Recommended):
    ```sh
    brew install go
    ```
  - Manual Installation:
    1. [Download Go](https://golang.org/dl/).
    2. Open the downloaded `.pkg` file and follow the instructions to install Go.

#### Install `lds`

Option A: Download the binary

1. Go to the [latest release](https://github.com/OpusMag/lds) and download the binary.
2. Make it executable:
   ```sh
   chmod +x ~/Downloads/lds
   ```
3. Move it to `/usr/local/bin/`:
   ```sh
   sudo mv ~/Downloads/lds /usr/local/bin/
   ```

Option B: Build from source

1. Clone the repository:
   ```sh
   git clone https://github.com/OpusMag/lds
   cd lds
   ```
2. Build the binary:
   ```sh
   go mod tidy
   go build -o lds main.go
   ```
3. Move it to `/usr/local/bin/`:
   ```sh
   sudo mv lds /usr/local/bin/
   ```

## Getting Started

To start `lds`, simply run: `lds`

## Usage

How to navigate, configure and change keybindings in lds:

## Navigation

- Move between boxes: Tab
- Navigate to parent directory: Press escape
- Navigate to sub directory: Highlight directory and press Enter (navigate to Directory box and use arrow keys)
- Highlight a file: Tab to the Files box and use the up and down arrow keys
- Open a file in an editor: highlight the file and hit Enter (highlighting can be done by searching or navigating to Files box and using arrow keys)

## Configuration

The configuration file should by default be located at ~/.config/lds/config.json. You can however have the config file wherever you want, but you have to add the path to configPath in main.go if you choose a different location than the defaults. In the config file you can customize colors, key bindings, and other settings.

## Key Bindings

- Quit: Ctrl+C
- Next Box: Tab
- Previous Box: Shift+Tab
- Go to parent directory: Press escape
- Go to sub directory: Highlight directory and press Enter
- Open a file in an editor : After highlighting a file, press Enter (highlighting can be done by searching or navigating to Files box and using arrow keys)
- Backspace: Backspace
- Rename: Alt+r
- Move: Alt+m
- Delete: Alt+d
- Copy: Alt+c

## Contributing

Contributions are welcome! Please open an issue or submit a pull request on GitHub.

## Support

For support, please open an issue on GitHub.

## License

This project is licensed under the MIT License.

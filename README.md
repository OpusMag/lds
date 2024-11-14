# lds
An alternative to the ls command that makes it easier to get your bearings in a complicated directory with lots of files

![lds](https://github.com/user-attachments/assets/f38c2770-5e40-4886-b6b0-4e444f298942)


To use this tool:

Install dependencies first (Git and Go programming language).

Linux: Depending on your package manager:

    Git:
        sudo pacman -S git -y
        sudo dnf install git-all
        sudo apt install git-all

    Go:
        sudo pacman -S go
        sudo dnf install go
        sudo apt install go

Windows:

    Git:
        go to https://git-scm.com/download/win and the download should start automatically. Then install it.

    Go:
        go to https://golang.org/dl/ and download the latest version of go.
        Follow this guide to install properly: https://www.geeksforgeeks.org/how-to-install-go-on-windows/

macOS

    Git:
        Using Homebrew (Recommended):

        brew install git

Using Xcode Command Line Tools:

xcode-select --install

Go:

    Using Homebrew (Recommended):

    brew install go

        Manual Installation:
            Download the latest version of Go from https://golang.org/dl/
            Open the downloaded .pkg file and follow the instructions to install Go.

Linux and macOS instructions for installing lds:

Option A:

    Go to the latest release at https://github.com/OpusMag/lds and download the binary.
    Eun chmod +x ~/Downloads/lds (or whatever dir you downloaded it to)
    Move it to /usr/local/bin/ by running sudo mv lds /usr/local/bin/

Option B:

    Enter your terminal.

    Git clone this repository either by using the URL or SSH (you can find this by clicking the green button in the upper right that says '<>Code').

    Then cd (change directory) into the repository.

    Run the following commands to build and run the CLI:

    go mod tidy go build -o lds main.go ./lds

    If you want to use it without the ./ prefix you can do the following: while in the lds directory and after building the binary: sudo mv lds /usr/local/bin/

    Confirm it has worked: lds --help

Windows instructions:

Option A:

    Go to the latest release at https://github.com/OpusMag/lds, download the binary lds.exe.
    Move the binary to a directory, for example C:\Tools.
    Add the directory to PATH by pressing Win + X and selecting System, then click on Advanced system settings.
    In the System properties window, click on the Environment variables button.
    In the Environment Variables window, find the Path variable in the System variables section and select it and click Edit.
    In the Edit Environment Variable window, click New and add the path to the directory where you placed lds.exe (for example c:\Tools), then click OK to close.
    Verify it's working by opening a terminal and running 'lds --help'.

Option B:

    'git clone https://github.com/OpusMag/lds' then cd into the repository.
    Run 'go mod tidy' in the terminal to make sure dependencies are installed.
    Then run 'go build -o what-cmd.exe main.go'. Then run the tool with '.\lds.exe'

General use of the tool(both windows and linux):

    You move between boxes by pressing tab. When you're in the search box, you can either search for a directory or file or press .. and then hit enter to navigate to the parent directory. If you move to the file box and highlight a file by scrolling up or down using the arrow keys, a new box appears that allows you to input a command you can perform on the highlighted file. If you move to the directory box and highlight a directory and then hit enter, you automatically navigate to that directory.

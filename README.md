# Varnam 

Varnam is an Indian language transliteration library. GoVarnam is a Go port of [libvarnam](https://github.com/varnamproject/libvarnam) with some core architectural changes. Not every part of libvarnam is ported.

It is stable to use daily as an input method. See it in action here: https://varnamproject.github.io/editor/

An [Input Method Engine](https://en.wikipedia.org/wiki/Input_method) for Linux operating systems via IBus is available here: https://github.com/varnamproject/govarnam-ibus

![](https://varnamproject.github.io/_index/free-to-write-anything.png)

## Installation

You will need to install GoVarnam library in your system for any app to use Varnam.

* Download a [recent GoVarnam version](https://github.com/varnamproject/govarnam/releases).
* Extract the zip file
* Open a terminal and go to the extracted folder by using this command :
```bash
cd Downloads/govarnam
```
* Now run this command to install GoVarnam :
```bash
sudo ./install.sh install
```
It will ask for your password, enter it.
* Basic Installation is finished

* Install your language from [here](https://github.com/varnamproject/schemes)

You may also install the IBus engine to use Varnam system wide: https://github.com/varnamproject/govarnam-ibus

## Usage

Test it out:
```bash
varnamcli -s ml namaskaaram
```

Learn a word:
```bash
varnamcli -s ml -learn കുന്നംകുളം
```

Train a word with a particular pattern:
```bash
varnamcli -s ml -train college കോളേജ്
```

### Learning Words From A File

You can import all language words from any text file. Varnam will separate english words and non-english words and learn accordingly.

```bash
varnamcli -s ml -learn-from-file file.html
```

You can download news articles or Wikipedia pages in HTML format to learn words from them.

### Export Learnings

You can export your local learnings with:
```bash
varnamcli -s ml -export my-words
```
The file extension will be `.vlf` [Varnam Learnings File]

### Import Learnings

You can import learnings from a `.vlf` :
```bash
varnamcli -s ml -import my-words-1.vlf
```

## Development

### Build

This repository have 3 things :

1. GoVarnam library
2. GoVarnam Command Line Utility (CLI)
3. Go bindings for GoVarnam

GoVarnam is written in Go, but to be a standard library that can be used with any other programming languages, we compile it to a C library. This is done by :
```bash
go build -buildmode "c-shared" -o libgovarnam.so
```

(Shortcut to doing above is `make library`)

The output `libgovarnam.so` is a shared library that can be dynamically linked in any other programming languages. Some examples :

* Go bindings for GoVarnam: See govarnam**go** folder in this repo
* Java bindings for GoVarnam: IN PROGRESS

Wait, it means we need to write another Go file to interface with GoVarnam library ! This is because we're interfacing with a shared library and not the Go library.

### Files & Folders

* `govarnam` - The library files
* `main.go, c-shared*` - Files that help in making the govarnam a C shared library
* `govarnamgo` - Go bindings for the library. For use with other Go projects
* `cli` - A CLI tool for varnam. Uses `govarnamgo` to interface with the library.
* `symbol-frequency-calculator` - For populating the `weight` column in VST files

### CLI (Command Line Utility)

The command line utility (CLI) is written in Go, uses govarnamgo to interface with the library.

You need to separately build the CLI:
```
cd cli

# Show the path to libgovarnam.so
export LD_LIBRARY_PATH=$(realpath ../):$LD_LIBRARY_PATH

go build -o varnamcli .
```

### Hacking

This section is straight on getting your hands in. Explanation of how GoVarnam works is at the bottom.

* Clone of course
* Do `go get`
* You will need a `.vst` file. Get it from `schemes` folder in [a release](https://github.com/varnamproject/govarnam-ibus/releases). Paste it in `schemes` folder
* Do `make library` to compile

When you make changes to govarnam source code, you will need to do `make library` for the changes to build on and then test with CLI.

You can run tests (to make sure nothing broke) with :
```bash
make test
```

## GoVarnam BTS

Read GoVarnam Spec: https://docs.google.com/document/d/1l5cZAkly_-kl7UkfeGmObSam-niWCJo4wq-OvAEaDvQ/edit?usp=sharing

### Changes from libvarnam

* `ml.vst` has been changed to add a new `weight` column in `symbols` table. Get the new `ml.vst` here. The symbol with the least weight has more significance. This is calculated according to popularity from corpus. You can populate a `ml.vst` with weight values by a Python script. See that in the subfolder. The previous ruby script is used for making the VST. That is the same. **`ml.vst` from libvarnam is incompatible with govarnam**.

* `patterns_content` is renamed to `patterns` in GoVarnam

* `patterns` table in learnings DB won't store malayalam patterns. Instead, for each input, all possible malayalam words are calculated (from `symbols` VARNAM_MATCH_ALL) and searched in `words`. These are returned as suggestions. Previously, `pattern` would store every pattern to a word. english => malayalam.

* `patterns` in govarnam is used solely for English words. `Computer => കമ്പ്യൂട്ടർ`. These English words won't work out with our VST tokenizer cause the words are not really transliterable in our language. It would be `kambyoottar => Computer`

## Miscellaneous

To build without SQLite :
```
go build -tags libsqlite3 -buildmode=c-shared -o libgovarnam.so
```

### Release Process

* git tag
* make build release

Pack ibus engine:
* make build-ubuntu18 release
# Varnam 

Varnam is an Indian language transliteration library. GoVarnam is a brand new Go port of [libvarnam](https://github.com/varnamproject/libvarnam) with some core architectural changes.

It is stable to use daily as an input method. Try out different languages here: https://varnamproject.github.io/editor/

Malayalam has really good support in Varnam. We welcome improvements of all languages in Varnam.

* An [Input Method Engine](https://en.wikipedia.org/wiki/Input_method) for GNU/Linux operating systems via IBus is available here: https://github.com/varnamproject/govarnam-ibus
* For macOS, there is a [Varnam IME too](https://github.com/varnamproject/varnam-macOS).
* Windows: Need Help

## Installation & Usage

See instructions in website: https://varnamproject.github.io/download/

FAQ: https://varnamproject.github.io/docs/faq/

<br/>

![](https://varnamproject.github.io/_index/free-to-write-anything.png)

## Development

Proceed through these sections one by one:

### Videos

See this video to understand more about Varnam (DebConf21):

* PeerTube: https://peertube.debian.social/w/vWwMGcmTZG9n1UWv8ZdimB?s=1
* YouTube: https://www.youtube.com/watch?v=pJpOWlD_7OI

### Files & Folders

* `govarnam` - The library files
* `main.go, c-shared*` - Files that help in making the govarnam a C shared library
* `govarnamgo` - Go bindings for the library. For use with other Go projects
* `cli` - A CLI tool written in Go for Varnam. Uses `govarnamgo` to interface with the library.

### Build Library

Requires minimum Go version 1.16.

This repository have 3 things :

1. GoVarnam library
2. GoVarnam Command Line Utility (CLI)
3. Go bindings for GoVarnam

GoVarnam is written in Go, but to be a standard library that can be used with any other programming languages, we compile it to a C library. This is done by :
```bash
go build -buildmode "c-shared" -o libgovarnam.so
```

(Shortcut to doing above is `make library`)

The output `libgovarnam.so` is a shared library that can be dynamically linked in any other programming languages using its header file `libgovarnam.h`. Some examples :

* Go bindings for GoVarnam: See govarnam**go** folder in this repo
* Java bindings for GoVarnam: https://github.com/varnamproject/govarnam-java/

Wait, it means we need to write another Go file to interface with GoVarnam library ! This is because we're interfacing with a C shared library and not the Go library directly. The `govarnamgo` acts as this interface for Go apps to use GoVarnam.

### CLI (Command Line Utility)

After making `libgovarnam.so` you can make the CLI to use GoVarnam :

```
make cli
```

The command line utility (CLI) is written in Go, uses govarnamgo to interface with the library.

You can build both library and CLI with just `make`.

### Language Support

Varnam uses a `.vst` (Varnam Symbol Table) file for language support. You can get it from it from `schemes` folder in [a release](https://github.com/varnamproject/schemes/releases). Place VST files in **one of these** locations (from high priority to least priority locations):

* `$PWD/schemes` (PWD is Present Working Directory)
* `/usr/local/share/varnam/schemes`
* `/usr/share/varnam/schemes`

Now we can use `varnamcli`:

```
# Show linker the path to search for libgovarnam.so
export LD_LIBRARY_PATH=$(realpath ./):$LD_LIBRARY_PATH

./varnamcli -s ml namaskaaram
```

The `ml` above is the scheme ID. It should match with the VST filename.

You can link the library to `/usr/local/lib` to skip doing the `export LD_LIBRARY_PATH` every time:

```
sudo ln -s $PWD/libgovarnam.so /usr/local/lib/libgovarnam.so
```

Now any software can find the GoVarnam library.

### Testing

You can run tests (to make sure nothing broke) with :
```bash
make test
```

### Use Varnam Live

It's good to install an IME to test changes you make to the library live.

* Linux IME: https://github.com/varnamproject/govarnam-ibus
* Mac IME (Coming Soon...): https://github.com/varnamproject/govarnam/issues/8
* Windows IME (Coming Soon...): https://github.com/varnamproject/govarnam/issues/7

### Changes from libvarnam

* `ml.vst` has been changed to add a new `weight` column in `symbols` table. Get the new `ml.vst` here. The symbol with the least weight has more significance. This is calculated according to popularity from corpus. You can populate a `ml.vst` with weight values by a Python script. See that in the subfolder. The previous ruby script is used for making the VST. That is the same. **`ml.vst` from libvarnam is incompatible with govarnam**.

* `patterns_content` is renamed to `patterns` in GoVarnam

* `patterns` table in learnings DB won't store malayalam patterns. Instead, for each input, all possible malayalam words are calculated (from `symbols` VARNAM_MATCH_ALL) and searched in `words`. These are returned as suggestions. Previously, `pattern` would store every pattern to a word. english => malayalam.

* `patterns` in govarnam is used solely for English words. `Computer => കമ്പ്യൂട്ടർ`. These English words won't work out with our VST tokenizer cause the words are not really transliterable in our language. It would be `kambyoottar => Computer`

# GoVarnam 

A transliteration shared library. A Go port of libvarnam with some core architectural changes.

## Files & Folders

* `govarnam` - The library files
* `main.go, c-shared*` - Files that help in making the govarnam a C shared library
* `govarnamgo` - Go bindings for the library. For use with other Go projects
* `cli` - A CLI tool for varnam. Uses `govarnamgo` to interface with the library.
* `symbol-frequency-calculator` - For populating the `weight` column in VST files

## Development

Download files from [here](https://mega.nz/folder/JnhmRDDI#MoVHlxkCh-1QR3Hxc8OcFQ). Copy files to these locations:
```
sudo mkdir -p /usr/local/share/varnam/vstDEV
sudo ln -s ml.vst /usr/local/share/varnam/vstDEV

mkdir -p ~/.local/share/varnam/suggestionsDEV
cp ml.vst.learnings ~/.local/share/varnam/suggestionsDEV
```

The first file is Varnam Symbol Table (VST) which has language information like consonants, vowels, combinations etc. Allows to tokenize a english word into Indian language.

The second file is a local data storage (Varnam Learnings File) that stores words. When varnam learns a new word it is stored in this.

Both VST & Learnigns file is an **SQLite database**.

### Build

Now clone this repo and build `govarnam` shared library :
```
go build -buildmode=c-shared -o libgovarnam.so
```

To build without SQLite :
```
go build -tags libsqlite3 -buildmode=c-shared -o libgovarnam.so
```

Two files will be generated: `libgovarnam.h` and `libgovarnam.so`.

## CLI (Command Line Utility)

You need to separately build the CLI:
```
cd cli
export LD_LIBRARY_PATH=$(realpath ../):$LD_LIBRARY_PATH
go build -o varnam .
```

### Usage

Test it out:
```
./varnam -lang ml namaskaaram
```

Learn a word:
```
./varnam -lang ml -learn കുന്നംകുളം
```

Train a word with a particular pattern:
```
./varnam -lang ml -train college കോളേജ്
```

## Changes from libvarnam

* `ml.vst` has been changed to add a new `weight` column in `symbols` table. Get the new `ml.vst` here. The symbol with the least weight has more significance. This is calculated according to popularity from corpus. You can populate a `ml.vst` with weight values by a Python script. See that in the subfolder. The previous ruby script is used for making the VST. That is the same. **`ml.vst` from libvarnam is incompatible with govarnam**.

* `patterns_content` table in learnings DB won't store malayalam patterns. Instead, for each input, all possible malayalam words are calculated (from `symbols` VARNAM_MATCH_ALL) and searched in `words`. These are returned as suggestions. Previously, `patterns_content` would store every pattern to a word. english => malayalam.

* `patterns_content` in govarnam is used solely for English words. `Computer => കമ്പ്യൂട്ടർ`. These English words won't work out with our VST tokenizer cause the words are not really transliterable in our language. It would be `kambyoottar => Computer`

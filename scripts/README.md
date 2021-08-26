# Scripts

## Remove symbols from word frequency report

This script removes VST symbols from a word frequency report file.

Sometimes frequency report will have items like "ലൂ", "ഓ" which is unnecessary because tokenizer can make these on its own.

GoVarnam won't learn single conjuncts anyway, so why keep it in report ? Remove them with :

```bash
python3 frequency-report-remove-symbols.py scheme.vst wordFrequencyReport.txt outputFile.txt
```

### Word Frequency Report File

A word frequency report file has this format :
```
word frequency
```
Example:
```
എന്ത് 14569
ഇത് 2045
വർഗ്ഗം 254
എന്ന 254
ഒരു 254
താളിലേക്ക് 254
ഫലകം 254
```

This file is made from analysing usage of words in internet. [This repo](https://github.com/AI4Bharat/indicnlp_corpus#text-corpora) has a premade vocab frequency file for some Indian languages. [Indic Keyboard](https://gitlab.com/indicproject/dictionaries) also has one.

### Normalize Frequency

A frequency report may have large difference between the first word and last word like this :

```
എന്ത് 14569000
...
...
ഫലകം 254
```

This is bad because suggestions can come out wrong. We need to normalize these values between a min and max.

To normalize frequency of words between a min value (15) and max value (255), we can use this :

```
perl frequency-normalizer.pl frequencyReport.txt 15 255
```

## Populate weight column in VST

In GoVarnam's VST, we will have a weight for each possibility symbol. This is to make the tokenizer output better for possible suggestions. More a symbol is in popular usage, the more that word have greater weight in tokenizer output.

* Get a word frequency report file (explained at the top of this README)

Such a file helps to calculate symbol frequency very easy. We just need to make a hashmap of each symbols in a word and add the corresponding word frequency value.

After we go through the entire list of words, we will have a hashmap of symbol frequency.

* Make the symbol frequency report :
```
python3 symbol-frequency-maker.py scheme.vst word-frequency.txt symbol-frequency.txt
```

Now the output file will have a similar content:
```
അ 951134
എ 763499
വ 739865
നി 710719
ക 500238
രു 478358
```

* Normalize the frequency :

```
perl frequency-normalizer.pl symbol-frequency.txt 0 100 > symbol-frequency-normalized.txt
```

* Now, use this file to populate the weight column in VST :
```
python3 symbol-weight-update-in-vst.py ml.vst symbol-frequency-normalized.txt
```

import sqlite3
import sys

desc = '''
Usage: script.py scheme.vst wordReportFile.txt outputSymbolReportFile.txt

INPUT FILE MUST BE WORD FREQUENCY REPORT. FORMAT:
word<space>frequency
എന്ത് 14569
ഇത് 2045
'''

if len(sys.argv) != 4:
    print(desc)
    sys.exit(0)

db = sys.argv[1]
file = sys.argv[2]
outputFile = sys.argv[3]

con = sqlite3.connect(db)
cur = con.cursor()

cur.execute("SELECT pattern, value1 FROM symbols WHERE pattern IN (SELECT pattern from symbols GROUP by pattern HAVING COUNT(pattern) > 1)")
patternsAndSymbols = cur.fetchall()
symbols = []
for s in patternsAndSymbols:
    symbols.append(s[1])

freqs = {}


def add(char, frequency):
    if char in freqs:
        freqs[char] += int(frequency)
    else:
        freqs[char] = int(frequency)
    # print("Incremented %s - %s" % (char, frequency))


base = 0
with open(file, "r", encoding="utf8", errors='ignore') as f:
    processedCount = 0
    for line in f:
        word, frequency = line.split(" ")

        i = 0
        sequence = ""
        while i < len(word):
            sequence += word[i]

            if sequence not in symbols:
                # backtrack
                if sequence[0:-1] in symbols:
                    add(sequence[0:-1], frequency)
                    sequence = sequence[-1]
            else:
                if i == len(word) - 1:
                    # Last character
                    add(sequence, frequency)

            i += 1
        
        processedCount += 1

        if processedCount % 30 == 0:
            print("Processed " + str(processedCount) + " words")

freqs = dict(sorted(freqs.items(), key=lambda item: item[1], reverse=True))

with open(outputFile, 'a+') as out:
    for grapheme, weight in freqs.items():
        outLine = grapheme + " " + str(weight)
        out.write(outLine + '\n')

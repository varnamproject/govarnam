import sqlite3
import sys

desc = '''
Usage: script.py scheme.vst inputFile.txt outputFile.txt

This script removes VST symbol words from a word frequency report file.
Sometimes frequency report will have items like "ലൂ", "ഓ" which
is unnecessary because tokenizer have these and can make these.

GoVarnam won't learn single conjuncts anyway, so why keep it in report ?

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

cur.execute("SELECT value1 FROM symbols")
value1 = cur.fetchall()
symbols = []
for s in value1:
    symbols.append(s[0])

base = 0
with open(file, "r", encoding="utf8", errors='ignore') as fileInput:
    with open(outputFile, "w") as fileOutput:
        processedCount = 0

        for line in fileInput:
            word, frequency = line.split(" ")

            if word in symbols:
                continue

            fileOutput.write(line)
            
            processedCount += 1

            if processedCount % 1000 == 0:
                print("Processed " + str(processedCount) + " lines")


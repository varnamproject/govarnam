import sqlite3
import sys

'''
INPUT FILE MUST BE WORD FREQUENCY REPORT. FORMAT:
word<space>frequency
എന്ത് 14569
ഇത് 2045
'''


db = sys.argv[1]
file = sys.argv[2]

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


base = 0
with open(file, "r", encoding="utf8", errors='ignore') as f:
    for line in f:
        word, frequency = line.split(" ")

        sequence = ""
        for char in word:
            sequence += char

            if sequence not in symbols:
                # backtrack
                if sequence[0:-1] in symbols:
                    add(sequence[0:-1], frequency)
                    sequence = ""
                    # print("Processed %s %s" % (sequence[0:-1], frequency))

freqs = dict(sorted(freqs.items(), key=lambda item: item[1], reverse=True))
for grapheme, weight in freqs.items():
    print(grapheme + " " + str(weight))

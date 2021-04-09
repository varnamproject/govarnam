import sqlite3
import sys


db = sys.argv[1]
file = sys.argv[2]

con = sqlite3.connect(db)
cur = con.cursor()

lang = list(map(chr, list(range(0x0D00, 0xD7F + 1)) + [0x200C, 0x200D]))
cur.execute("SELECT * FROM symbols WHERE pattern IN (select pattern from symbols GROUP by pattern HAVING COUNT(pattern) > 1)")
symbols = []
for s in cur.fetchall():
    symbols.append(s[0])

freqs = {}


def add(char):
    if char in freqs:
        freqs[char] += 1
    else:
        freqs[char] = 1


i = 1
with open(file, "r", encoding="utf8", errors='ignore') as f:
    for line in f:
        sequence = ""
        for char in line:
            if char in lang:
                sequence += char
            elif sequence == "":
                continue

            if sequence not in symbols:
                # backtrack
                if sequence[0:-1] in symbols:
                    add(sequence[0:-1])
                sequence = ""
        # print("Processed %d lines" % i)
        # i += 1

freqs = dict(sorted(freqs.items(), key=lambda item: item[1], reverse=True))
for k, v in freqs.items():
    print(k, v)

import sqlite3
import sys

'''
INPUT FILE MUST BE SYMBOL FREQUENCY REPORT. FORMAT:
symbol<space>frequency
സ 2045414
യെ 1456
'''

db = sys.argv[1]
file = sys.argv[2]

con = sqlite3.connect(db)
cur = con.cursor()

cur.execute("SELECT pattern, value1 FROM symbols WHERE pattern IN (SELECT pattern from symbols GROUP by pattern HAVING COUNT(pattern) > 1)")
patternsAndSymbols = cur.fetchall()

freqs = {}
with open(file, "r", encoding="utf8", errors='ignore') as f:
    for line in f:
        word, frequency = line.split(" ")
        freqs[word] = int(frequency)

patternAndSymbols = {}
for pattern, symbol in patternsAndSymbols:
    if pattern not in patternAndSymbols:
        patternAndSymbols[pattern] = [(symbol, freqs[symbol] if symbol in freqs else 0)]
    else:
        patternAndSymbols[pattern].append((symbol, freqs[symbol] if symbol in freqs else 0))

for pattern, symbols in patternAndSymbols.items():
    ranks = {}

    s = 0
    for symbol, freq in symbols:
        s += freq

    if s == 0:
        continue
    
    for symbol, freq in symbols:
        ranks[symbol] = int((int(freq) / s) * 100)

    ranks = dict(sorted(ranks.items(), key=lambda item: item[1], reverse=True))

    for symbol, rank in ranks.items():
        print(pattern, rank, symbol)

        cur.execute("UPDATE symbols SET weight = ? WHERE pattern = ? AND value1 = ?", (rank, pattern, symbol))

        rank += 1
con.commit()

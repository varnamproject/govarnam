import sqlite3
import sys

desc = '''
Usage: script.py scheme.vst symbolFrequencyReport.txt

INPUT FILE MUST BE SYMBOL FREQUENCY REPORT. FORMAT:
symbol<space>frequency
സ 2045414
യെ 1456
'''

if len(sys.argv) != 3:
    print(desc)
    sys.exit(0)

db = sys.argv[1]
file = sys.argv[2]

con = sqlite3.connect(db)
cur = con.cursor()

cur.execute("SELECT pattern, value1, type FROM symbols WHERE match_type = 2 AND pattern IN (SELECT pattern from symbols GROUP by pattern HAVING COUNT(pattern) > 1)")
patternsAndSymbols = cur.fetchall()

freqs = {}
with open(file, "r", encoding="utf8", errors='ignore') as f:
    for line in f:
        symbol, frequency = line.split(" ")
        freqs[symbol] = int(frequency)

patternAndSymbols = {}
for pattern, symbol, symbolType in patternsAndSymbols:
    if pattern not in patternAndSymbols:
        patternAndSymbols[pattern] = [(
            symbol,
            symbolType,
            freqs[symbol] if symbol in freqs else 0
        )]
    else:
        patternAndSymbols[pattern].append((
            symbol,
            symbolType,
            freqs[symbol] if symbol in freqs else 0
        ))

for pattern, symbols in patternAndSymbols.items():
    CONSONANT = 2  # ണ്ട
    CONSONANT_VOWEL = 4  # ണ്ടാ

    ranks = {}

    # Find the consonant with least frequency value
    minConsonantFrequency = 100
    maxConsonantVowelFrequency = 1
    for symbol, symbolType, frequency in symbols:
        if symbolType == CONSONANT and minConsonantFrequency > frequency:
            minConsonantFrequency = frequency

        if symbolType == CONSONANT_VOWEL and maxConsonantVowelFrequency < frequency:
            maxConsonantVowelFrequency = frequency

        if symbolType != CONSONANT_VOWEL:
            ranks[symbol] = frequency

    for symbol, symbolType, frequency in symbols:
        if symbolType == CONSONANT_VOWEL:
            ranks[symbol] = int((frequency / maxConsonantVowelFrequency) * (minConsonantFrequency / 2))

    ranks = dict(sorted(ranks.items(), key=lambda item: item[1], reverse=True))

    for symbol, rank in ranks.items():
        print(pattern, rank, symbol)

        cur.execute("UPDATE symbols SET weight = ? WHERE pattern = ? AND value1 = ?", (rank, pattern, symbol))

        rank += 1
con.commit()

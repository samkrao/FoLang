"""Check: a decision row must not cite a grammar production that no longer
exists, UNLESS the row is recording that production's removal."""
import re, sys
sys.path.insert(0,'/home/claude/fo')
from check import parse

REMOVAL = re.compile(r'\b(removed|withdraw|withdrawn|superseded|deleted|no longer)\b', re.I)
NAME    = re.compile(r'`([a-z][a-z0-9]*(?:-[a-z0-9]{2,})+)`')

def run(grammar, register):
    _, P, _ = parse(grammar)
    reg = open(register, encoding='utf-8').read()
    bad = []
    for row in re.findall(r'(?m)^\| `(DECISION-[A-Z]+-\d+)`.*$', reg):
        pass
    for m in re.finditer(r'(?m)^\| `(DECISION-[A-Z]+-\d+)` \|(.*)$', reg):
        did, body = m.group(1), m.group(2)
        for tok in set(NAME.findall(body)):
            if tok in P: continue
            if REMOVAL.search(body): continue      # legitimate withdrawal note
            bad.append((did, tok))
    return bad

if __name__ == '__main__':
    bad = run('folang.ebnf','grammar-decisions.md')
    if bad:
        for d,t in bad: print(f"  DANGLING  {d} cites `{t}` which is not a production")
    else:
        print("  OK  no decision cites a nonexistent production outside a removal note")

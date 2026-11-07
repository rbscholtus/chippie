import hashlib, json, os, sys, collections

A = sys.argv[1]
db = os.path.join(A, "chip-8-database/database")
hashes = json.load(open(os.path.join(db, "sha1-hashes.json")))
programs = json.load(open(os.path.join(db, "programs.json")))
platforms = {p["id"]: p["name"] for p in json.load(open(os.path.join(db, "platforms.json")))}

sources = {
    "Timendus test suite": ("chip8-test-suite/bin", lambda f: f.endswith(".ch8")),
    "chip8Archive (J. Earnest)": ("chip8Archive/roms", lambda f: True),
    "kripod/chip8-roms": ("chip8-roms", lambda f: not f.lower().endswith((".txt", ".md", ".gitignore"))),
    "Zophar packs": ("zophar", lambda f: not f.endswith(".bin") and not f.lower().endswith((".txt", ".doc"))),
    "c8games (attached)": ("c8games", lambda f: not f.lower().endswith((".txt", ".doc"))),
}

print(f"Database: {len(programs)} programs, {len(hashes)} ROM hashes, {len(platforms)} platforms\n")
all_hashes = {}
for name, (d, keep) in sources.items():
    root = os.path.join(A, d)
    seen, hit, miss = set(), 0, []
    for dp, dn, fn in os.walk(root):
        if ".git" in dp:
            continue
        for f in fn:
            if not keep(f):
                continue
            p = os.path.join(dp, f)
            data = open(p, "rb").read()
            if len(data) == 0 or len(data) > 65536:
                continue
            h = hashlib.sha1(data).hexdigest()
            if h in seen:
                continue
            seen.add(h)
            all_hashes.setdefault(h, set()).add(name)
            if h in hashes:
                hit += 1
            else:
                miss.append(os.path.relpath(p, root))
    n = len(seen)
    print(f"{name:28s} unique ROMs {n:4d}  in DB {hit:4d} ({100*hit/max(n,1):5.1f}%)  missing {len(miss)}")
    if miss and "-v" in sys.argv:
        for m in sorted(miss):
            print("    -", m)

tot = len(all_hashes)
hit = sum(1 for h in all_hashes if h in hashes)
print(f"\nAll sources combined: {tot} unique ROMs, {hit} in DB ({100*hit/tot:.1f}%), {tot-hit} missing")
overlap = collections.Counter(len(v) for v in all_hashes.values())
print("Unique ROMs by number of sources carrying them:", dict(sorted(overlap.items())))

# platform distribution of DB
pc = collections.Counter()
for prog in programs:
    for r in prog["roms"].values():
        for pl in r.get("platforms", [])[:1]:
            pc[pl] += 1
print("\nDB ROMs by primary platform:")
for k, v in pc.most_common():
    print(f"  {platforms.get(k, k):35s} {v}")

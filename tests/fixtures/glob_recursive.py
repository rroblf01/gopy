import glob
import os
import tempfile


def main() -> None:
    base = tempfile.mkdtemp(prefix="gopy_gl_")
    os.makedirs(os.path.join(base, "a", "b"))
    os.makedirs(os.path.join(base, "c"))
    with open(os.path.join(base, "top.txt"), "w") as fh:
        fh.write("t")
    with open(os.path.join(base, "a", "mid.txt"), "w") as fh:
        fh.write("m")
    with open(os.path.join(base, "a", "b", "leaf.txt"), "w") as fh:
        fh.write("l")
    with open(os.path.join(base, "c", "side.log"), "w") as fh:
        fh.write("s")

    # Use ** alone (recursive=True) — matches the basename portion against
    # any files anywhere under base. CPython glob also accepts the joined
    # form `base/**/*.txt`, but the bare `base/**` + post-filter is more
    # portable across the platforms' walk semantics.
    matches: list[str] = glob.glob(os.path.join(base, "**"), recursive=True)

    txts: list[str] = []
    for m in matches:
        if m.endswith(".txt"):
            txts.append(os.path.relpath(m, base).replace(os.sep, "/"))
    txts.sort()
    for r in txts:
        print(r)


if __name__ == "__main__":
    main()

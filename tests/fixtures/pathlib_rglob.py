import os
import tempfile
from pathlib import Path


def main() -> None:
    base = tempfile.mkdtemp(prefix="gopy_rglob_")
    os.makedirs(os.path.join(base, "a", "b"))
    os.makedirs(os.path.join(base, "a", "c"))
    Path(os.path.join(base, "top.txt")).write_text("t")
    Path(os.path.join(base, "a", "mid.txt")).write_text("m")
    Path(os.path.join(base, "a", "b", "leaf.txt")).write_text("l")
    Path(os.path.join(base, "a", "c", "skip.log")).write_text("s")

    matches: list[str] = []
    for p in Path(base).rglob("*.txt"):
        matches.append(os.path.relpath(str(p), base))
    matches.sort()
    for m in matches:
        # Print with forward slashes so Linux/macOS output matches.
        print(m.replace(os.sep, "/"))


if __name__ == "__main__":
    main()

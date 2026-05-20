import os
from pathlib import Path


def main() -> None:
    base = Path("/tmp/gopy_glob_test")
    os.makedirs("/tmp/gopy_glob_test", exist_ok=True)
    a = base / "a.txt"
    b = base / "b.txt"
    c = base / "c.log"
    a.write_text("x")
    b.write_text("y")
    c.write_text("z")
    names: list[str] = []
    for child in base.glob("*.txt"):
        names.append(child.name)
    names.sort()
    for n in names:
        print(n)
    a.unlink()
    b.unlink()
    c.unlink()


if __name__ == "__main__":
    main()

import os
import tempfile


def main() -> None:
    d: str = tempfile.mkdtemp(prefix="gopy_scan_")
    with open(os.path.join(d, "a.txt"), "w") as f:
        f.write("hello")
    with open(os.path.join(d, "b.txt"), "w") as f:
        f.write("world")
    names: list[str] = []
    with os.scandir(d) as it:
        for e in it:
            names.append(e.name)
    names.sort()
    for n in names:
        print(n)
    # Cleanup.
    for n in names:
        os.remove(os.path.join(d, n))
    os.rmdir(d)


if __name__ == "__main__":
    main()

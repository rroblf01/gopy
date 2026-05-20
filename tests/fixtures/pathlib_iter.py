from pathlib import Path


def main() -> None:
    f1 = Path("/tmp/gopy_pl_a.txt")
    f2 = Path("/tmp/gopy_pl_b.log")
    f1.write_text("hi")
    f2.write_text("hello")
    print(f1.suffix)
    print(f1.stem)
    print(f2.suffix)
    print(f2.stem)
    print(f1.exists())
    f1.unlink()
    f2.unlink()
    print(f1.exists())


if __name__ == "__main__":
    main()

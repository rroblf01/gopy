from pathlib import Path


def main() -> None:
    p = Path("/tmp/gopy_pathlib_test.txt")
    p.write_text("hello path\n")
    print(p.exists())
    print(p.is_file())
    print(p.is_dir())
    print(p.read_text())
    bogus = Path("/tmp/__definitely_not_here_12345__")
    print(bogus.exists())


if __name__ == "__main__":
    main()

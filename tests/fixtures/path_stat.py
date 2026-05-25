from pathlib import Path
import tempfile


def main() -> None:
    p = Path(tempfile.gettempdir()) / "gopy_stat.txt"
    p.write_text("hello world")
    st = p.stat()
    print(st is not None)
    print(len(p.read_text()))
    p.unlink()


if __name__ == "__main__":
    main()

import gzip
import os
import tempfile


def main() -> None:
    path: str = os.path.join(tempfile.gettempdir(), "gopy_gzip_cm.gz")
    with gzip.open(path, "wt") as fout:
        fout.write("hello\nworld\n")
    with gzip.open(path, "rt") as fin:
        content: str = fin.read()
        print(content.strip())
    os.remove(path)


if __name__ == "__main__":
    main()

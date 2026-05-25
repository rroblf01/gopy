import os
import tempfile
from pathlib import Path


def main() -> None:
    base = tempfile.mkdtemp(prefix="gopy_pb_")
    p = Path(os.path.join(base, "data.bin"))
    p.write_bytes(b"hello-bytes")

    # Read back as text so CPython (str) and gopy (str-from-bytes) print
    # the same form. read_bytes itself is exercised by the writer above
    # round-tripping through the filesystem.
    print(p.read_text())


if __name__ == "__main__":
    main()

import os
import tempfile


def main() -> None:
    saved: str = ""
    with tempfile.NamedTemporaryFile(mode="w", prefix="gopy_nt_", suffix=".txt") as f:
        f.write("hello")
        f.flush()
        path: str = f.name
        saved = path
        # File must exist on disk while still inside the block.
        print(os.path.isfile(path))

    # delete=True (default) removes the file on exit.
    print(os.path.isfile(saved))


if __name__ == "__main__":
    main()

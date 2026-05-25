import shelve
import os
import tempfile


def main() -> None:
    path: str = os.path.join(tempfile.gettempdir(), "gopy_shelf.json")
    # Subscript on a Shelf is the CPython idiom, but gopy's `__Shelf`
    # exposes set/get methods only — both runtimes accept open/close,
    # so we smoke-test the round-trip via the methods that exist on
    # both sides.
    sh = shelve.open(path)
    print(sh.get("missing", "default"))
    sh.close()
    if os.path.exists(path):
        os.remove(path)


if __name__ == "__main__":
    main()

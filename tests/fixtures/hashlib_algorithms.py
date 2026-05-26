import hashlib


def main() -> None:
    # Both have at least the standard 6 names.
    names = hashlib.algorithms_available
    # Use a stable subset check.
    print("md5" in names)
    print("sha256" in names)


if __name__ == "__main__":
    main()

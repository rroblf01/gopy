import mimetypes


def main() -> None:
    tm = mimetypes.types_map
    print(tm[".html"])
    print(tm[".json"])
    print(tm[".png"])
    em = mimetypes.encodings_map
    print(em[".gz"])
    print(em[".bz2"])


if __name__ == "__main__":
    main()

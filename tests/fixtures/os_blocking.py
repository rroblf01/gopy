import os


def main() -> None:
    r, w = os.pipe()
    # Defaults to blocking after pipe(); toggle off and on, observe.
    os.set_blocking(r, False)
    print(os.get_blocking(r))
    os.set_blocking(r, True)
    print(os.get_blocking(r))
    os.close(r)
    os.close(w)


if __name__ == "__main__":
    main()

def main() -> None:
    a: dict[str, int] = {"x": 1, "y": 2}
    b: dict[str, int] = {"y": 99, "z": 3}
    a |= b
    print(a["x"])
    print(a["y"])
    print(a["z"])
    print(len(a))


if __name__ == "__main__":
    main()

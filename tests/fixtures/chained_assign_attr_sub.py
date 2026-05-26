class Box:
    value: int

    def __init__(self) -> None:
        self.value = 0


def main() -> None:
    a = Box()
    b = Box()
    a.value = b.value = 7
    print(a.value)
    print(b.value)

    xs = [0, 0, 0]
    ys = [0, 0, 0]
    xs[0] = ys[2] = 42
    print(xs)
    print(ys)

    d: dict[str, int] = {"x": 0, "y": 0}
    e: dict[str, int] = {"x": 0, "y": 0}
    d["x"] = e["y"] = 99
    print(d["x"], d["y"])
    print(e["x"], e["y"])


if __name__ == "__main__":
    main()

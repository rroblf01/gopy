from dataclasses import dataclass, asdict, astuple, replace


@dataclass
class Point:
    x: int
    y: int
    name: str = "p"


def main() -> None:
    p = Point(3, 4, "origin")
    d = asdict(p)
    print(d["x"])
    print(d["y"])
    print(d["name"])
    t = astuple(p)
    print(t[0])
    print(t[1])
    print(t[2])
    p2 = replace(p, x=99)
    print(p2.x)
    print(p2.y)
    print(p2.name)
    p3 = replace(p, name="moved", y=10)
    print(p3.x)
    print(p3.y)
    print(p3.name)


if __name__ == "__main__":
    main()

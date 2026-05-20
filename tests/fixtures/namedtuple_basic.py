from collections import namedtuple


Point = namedtuple("Point", ["x", "y"])
Person = namedtuple("Person", "name age")


def main() -> None:
    p = Point(3, 4)
    print(p.x)
    print(p.y)
    q = Person("ada", 36)
    print(q.name)
    print(q.age)


if __name__ == "__main__":
    main()

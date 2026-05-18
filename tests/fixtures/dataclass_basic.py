from dataclasses import dataclass


@dataclass
class Point:
    x: int
    y: int


@dataclass
class Person:
    name: str
    age: int = 25


def main() -> None:
    p: Point = Point(3, 4)
    print(p.x)
    print(p.y)
    q: Person = Person("ada")
    print(q.name)
    print(q.age)
    r: Person = Person("grace", 47)
    print(r.name)
    print(r.age)


if __name__ == "__main__":
    main()

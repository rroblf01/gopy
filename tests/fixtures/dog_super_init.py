class Animal:
    def __init__(self, name: str) -> None:
        self.name = name


class Dog(Animal):
    def __init__(self, name: str, breed: str) -> None:
        super().__init__(name)
        self.breed = breed

    def info(self) -> str:
        return f"{self.name} ({self.breed})"


def main() -> None:
    d = Dog("Rex", "Labrador")
    print(d.name)
    print(d.breed)
    print(d.info())


if __name__ == "__main__":
    main()

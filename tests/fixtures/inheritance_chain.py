class Vehicle:
    def __init__(self, wheels: int) -> None:
        self.wheels = wheels


class Car(Vehicle):
    def __init__(self, model: str) -> None:
        super().__init__(4)
        self.model = model


class SportsCar(Car):
    def __init__(self, model: str, top_speed: int) -> None:
        super().__init__(model)
        self.top_speed = top_speed


def main() -> None:
    c = Car("Sedan")
    print(c.model, c.wheels)
    s = SportsCar("Ferrari", 300)
    print(s.model, s.wheels, s.top_speed)


if __name__ == "__main__":
    main()

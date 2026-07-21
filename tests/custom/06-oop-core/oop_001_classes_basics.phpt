--TEST--
OOP 001: classes, constructors, visibility, static members, constants, inheritance, interfaces, abstract
--FILE--
<?php
class Animal {
    const KINGDOM = 'Animalia';
    public $name;
    protected $sound = '...';
    private $secret = 'hidden';
    public static $count = 0;

    public function __construct($name) {
        $this->name = $name;
        self::$count++;
    }

    public function speak() {
        return "$this->name says $this->sound";
    }

    protected function getSecretLength() {
        return strlen($this->secret);
    }
}

class Dog extends Animal {
    protected $sound = 'Woof';

    public function reveal() {
        return $this->getSecretLength();
    }
}

interface Shape {
    public function area();
}

abstract class BaseShape implements Shape {
    abstract public function area();
    public function describe() {
        return "Area: " . $this->area();
    }
}

class Square extends BaseShape {
    private $side;
    public function __construct($side) { $this->side = $side; }
    public function area() { return $this->side * $this->side; }
}

$d = new Dog('Rex');
var_dump($d->speak());
var_dump(Animal::$count);
$d2 = new Dog('Fido');
var_dump(Animal::$count);
var_dump(Dog::KINGDOM);
var_dump($d->reveal());
var_dump($d instanceof Animal);
var_dump($d instanceof Shape);

$sq = new Square(4);
var_dump($sq->describe());
var_dump($sq instanceof Shape);
var_dump($sq instanceof BaseShape);

var_dump(get_class($d));
var_dump(get_parent_class($d));
var_dump(method_exists($d, 'speak'));
var_dump(property_exists($d, 'name'));
?>
--EXPECT--
string(13) "Rex says Woof"
int(1)
int(2)
string(8) "Animalia"
int(6)
bool(true)
bool(false)
string(8) "Area: 16"
bool(true)
bool(true)
string(3) "Dog"
string(6) "Animal"
bool(true)
bool(true)

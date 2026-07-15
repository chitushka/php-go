--TEST--
P2 class and interface names are case-insensitive
--FILE--
<?php
class BaseCase {
    const KIND = 'base';
    public static $count = 1;

    public function value() {
        return 'v';
    }

    public static function bump() {
        self::$count = self::$count + 1;
        return self::$count;
    }
}

class ChildCase extends basecase {
}

interface NamedCase {
    const LABEL = 'named';
    public function name();
}

interface LabeledCase extends namedcase {
}

class UserCase implements labeledcase {
    public function NAME() {
        return 'Ada';
    }
}

$name = 'childcase';
$child = new $name();
echo $child instanceof CHILDCASE;
echo ':';
echo childcase::KIND;
echo ':';
echo CHILDCASE::BUMP();
echo ':';
echo $child instanceof BASECASE;
echo ':';
echo $child->VALUE();
echo "\n";

$user = new USERCASE();
echo $user instanceof NAMEDCASE;
echo ':';
echo labeledcase::LABEL;
echo ':';
echo USERCASE::LABEL;
echo ':';
echo $user->nAmE();
--EXPECT--
1:base:2:1:v
1:named:named:Ada

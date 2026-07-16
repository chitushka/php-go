--TEST--
P2 private parent members use declaring class scope
--FILE--
<?php
class BaseScope {
    private $value = 'base';

    public function baseValue() {
        return $this->value;
    }

    public function setBase($value) {
        $this->value = $value;
    }

    public function run() {
        return $this->secret();
    }

    private function secret() {
        return 'base-secret';
    }
}

class ChildScope extends BaseScope {
    public $value = 'child';

    public function childValue() {
        return $this->value;
    }

    public function secret() {
        return 'child-secret';
    }
}

$child = new ChildScope();
echo $child->baseValue();
echo ":";
echo $child->childValue();
echo ":";
echo $child->value;
echo "\n";
echo $child->run();
echo ":";
echo $child->secret();
echo "\n";
$child->setBase('new-base');
$child->value = 'new-child';
echo $child->baseValue();
echo ":";
echo $child->childValue();
echo ":";
echo $child->value;
?>
--EXPECT--
base:child:child
base-secret:child-secret
new-base:new-child:new-child

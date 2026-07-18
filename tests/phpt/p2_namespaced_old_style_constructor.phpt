--TEST--
P2 namespaced old-style constructor names are ordinary methods
--FILE--
<?php
namespace N;

class Box {
    public $value = 'default';

    function Box($value) {
        $this->value = $value;
        return 'called';
    }
}

$box = new Box();
echo $box->value;
echo "\n";
echo $box->Box('manual');
echo "\n";
echo $box->value;
echo "\n";

class StaticBox {
    static function StaticBox() {
        return 'static';
    }
}

echo StaticBox::StaticBox();
?>
--EXPECT--
default
called
manual
static

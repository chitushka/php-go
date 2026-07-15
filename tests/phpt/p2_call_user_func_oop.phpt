--TEST--
P2 call_user_func for OOP callables
--FILE--
<?php
function cuf_label($value) {
    return 'fn:' . $value;
}

class CufBox {
    public function run($value) {
        return 'method:' . $value;
    }

    public static function stat($value) {
        return 'static:' . $value;
    }

    public function __call($name, $args) {
        return 'magic:' . $name . ':' . $args[0];
    }

    public static function __callStatic($name, $args) {
        return 'magic-static:' . $name . ':' . $args[0];
    }

    public function __invoke($value) {
        return 'invoke:' . $value;
    }
}

$box = new CufBox();
$closure = function($value) {
    return 'closure:' . $value;
};

echo call_user_func('cuf_label', 'a');
echo ':';
echo call_user_func($closure, 'b');
echo ':';
echo call_user_func(array($box, 'run'), 'c');
echo ':';
echo call_user_func(array($box, 'missing'), 'd');
echo ':';
echo call_user_func('CufBox::stat', 'e');
echo ':';
echo call_user_func(array('CufBox', 'missingStatic'), 'f');
echo ':';
echo call_user_func($box, 'g');
--EXPECT--
fn:a:closure:b:method:c:magic:missing:d:static:e:magic-static:missingStatic:f:invoke:g

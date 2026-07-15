--TEST--
P2 call_user_func_array for OOP callables
--FILE--
<?php
function cufa_label($value) {
    return 'fn:' . $value;
}

class CufaBox {
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

$box = new CufaBox();
$closure = function($value) {
    return 'closure:' . $value;
};

echo call_user_func_array('cufa_label', array('a'));
echo ':';
echo call_user_func_array($closure, array('b'));
echo ':';
echo call_user_func_array(array($box, 'run'), array('c'));
echo ':';
echo call_user_func_array(array($box, 'missing'), array('d'));
echo ':';
echo call_user_func_array('CufaBox::stat', array('e'));
echo ':';
echo call_user_func_array(array('CufaBox', 'missingStatic'), array('f'));
echo ':';
echo call_user_func_array($box, array('g'));
echo ':';
echo call_user_func_array('strlen', array('abcd'));
--EXPECT--
fn:a:closure:b:method:c:magic:missing:d:static:e:magic-static:missingStatic:f:invoke:g:4

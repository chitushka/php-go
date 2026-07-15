--TEST--
P2 invokable objects and is_callable
--FILE--
<?php
class InvokableBox {
    public function __invoke($prefix, &$value) {
        $value = $value . ':changed';
        return $prefix . ':' . $value;
    }
}

class HiddenInvoke {
    private function __invoke() {}
}

$box = new InvokableBox();
$value = 'start';

echo $box('go', $value);
echo ':';
echo $value;
echo ':';
echo is_callable($box) ? 'callable' : 'no';
echo ':';
$hidden = new HiddenInvoke();
echo is_callable($hidden) ? 'hidden' : 'no';
echo ':';
echo is_callable($box, false, $name) ? $name : 'no';
--EXPECT--
go:start:changed:start:changed:callable:no:InvokableBox::__invoke

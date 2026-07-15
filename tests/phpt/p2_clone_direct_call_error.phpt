--TEST--
P2 __clone cannot be called directly
--FILE--
<?php
class Box {
    public function __clone() {
        echo 'clone';
    }
}

$box = new Box();
$box->__clone();
--EXPECTF--
cannot call __clone directly; use clone

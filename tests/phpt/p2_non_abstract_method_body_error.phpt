--TEST--
P2 non-abstract methods must contain body
--FILE--
<?php
class MissingBody {
    public function value();
}
?>
--EXPECTF--
parse errors: [Non-abstract method MissingBody::value() must contain body]

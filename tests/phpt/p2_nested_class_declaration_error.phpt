--TEST--
P2 nested class declarations are rejected
--FILE--
<?php
class Outer {
    function make() {
        class Inner {}
    }
}
?>
--EXPECTF--
parse errors: [Class declarations may not be nested]

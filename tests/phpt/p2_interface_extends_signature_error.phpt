--TEST--
P2 interface extends method cannot add required parameter
--FILE--
<?php
interface Named {
    public function name($first);
}

interface DetailedNamed extends Named {
    public function name($first, $last);
}
--EXPECTF--
method DetailedNamed::name cannot require 2 parameters to match Named::name requiring 1

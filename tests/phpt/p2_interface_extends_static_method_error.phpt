--TEST--
P2 interface extends static modifier must match
--FILE--
<?php
interface StaticParentContract {
    public static function name();
}

interface StaticChildContract extends StaticParentContract {
    public function name();
}
?>
--EXPECTF--
method StaticChildContract::name static modifier must match StaticParentContract::name

--TEST--
P2 abstract class interface static modifier must match when declared
--FILE--
<?php
interface AbstractStaticNamed {
    public static function name();
}

abstract class AbstractStaticUser implements AbstractStaticNamed {
    public function name() {}
}
?>
--EXPECTF--
method AbstractStaticUser::name static modifier must match AbstractStaticNamed::name

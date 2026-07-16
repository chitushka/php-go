--TEST--
P2 interface implementation static modifier must match
--FILE--
<?php
interface StaticContract {
    public static function name();
}

class StaticContractUser implements StaticContract {
    public function name() {
        return 'Ada';
    }
}
?>
--EXPECTF--
method StaticContractUser::name static modifier must match StaticContract::name

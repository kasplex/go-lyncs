/*
** GMP wrapper library for simple integer operations.
*/

////////////////////////////////
#include <gmp.h>

#include "lua.h"
#include "lauxlib.h"
#include "lualib.h"

#include "lj_lib.h"

////////////////////////////////
static int _gmp_nlimit = 0;
static mpz_t *_gmp_zlimit = NULL;
#define _GMP_ZLIMIT(z) if (_gmp_zlimit) mpz_tdiv_r(z, z, *_gmp_zlimit);
#define _GMP_ZMT "_gmp_zmt"
#define _GMP_ZCHK_(z,n) \
  mpz_t *z = (mpz_t*)luaL_checkudata(L, n, _GMP_ZMT);
#define _GMP_ZCHK_ab \
  mpz_t *a = (mpz_t*)luaL_checkudata(L, 1, _GMP_ZMT); \
  mpz_t *b = (mpz_t*)luaL_checkudata(L, 2, _GMP_ZMT);
#define _GMP_ZCHK_zab \
  mpz_t *z = (mpz_t*)luaL_checkudata(L, 1, _GMP_ZMT); \
  mpz_t *a = (mpz_t*)luaL_checkudata(L, 2, _GMP_ZMT); \
  mpz_t *b = (mpz_t*)luaL_checkudata(L, 3, _GMP_ZMT);
#define _GMP_ZCHK_limit(n,s) if (_gmp_nlimit && n>_gmp_nlimit) { luaL_error(L, s); return 0; }
#define _GMP_ZCHK_DIV_zero(z,s) if (0==mpz_sgn(*z)) { luaL_error(L, s); return 0; }

////////////////////////////////
void _gmp_zsetto(lua_State *L, mpz_t *z, int i) {
  if (lua_type(L, i)==LUA_TSTRING) {
    const char *s = luaL_checkstring(L, i);
    int b = luaL_optinteger(L, i+1, 0);
    if (b<2 || b>62) b = 0;
    if (mpz_set_str(*z, s, b) != 0) {
      luaL_error(L, "gmp_zset() failed");
      return;
    }
  } else if (lua_type(L, i)==LUA_TNUMBER) {
    double d = luaL_checknumber(L, i);
    mpz_set_d(*z, d);
  } else if (lua_type(L, i)==LUA_TNIL) {
    mpz_set_d(*z, 0);
  } else {
    _GMP_ZCHK_(a, i)
    mpz_set(*z, *a);
  }
  _GMP_ZLIMIT(*z);  
}

////////////////////////////////
static int gmp_znew(lua_State *L) {
  mpz_t *z = (mpz_t*)lua_newuserdata(L, sizeof(mpz_t));
  mpz_init(*z);
  _gmp_zsetto(L, z, 1);
  luaL_getmetatable(L, _GMP_ZMT);
  lua_setmetatable(L, -2);
  return 1;
}

////////////////////////////////
static int gmp_zclear(lua_State *L) {
  _GMP_ZCHK_(z, 1)
  mpz_clear(*z);
  return 0;
}

////////////////////////////////
static int gmp_zset(lua_State *L) {
  _GMP_ZCHK_(z, 1)
  _gmp_zsetto(L, z, 2);
  lua_pushvalue(L, 1);
  return 1;
}

////////////////////////////////
static int gmp_zstr(lua_State *L) {
  _GMP_ZCHK_(z, 1)
  int b = luaL_optinteger(L, 2, 0);
  if (b<2 || b>62) b = 10;
  char *s = mpz_get_str(NULL, b, *z);
  lua_pushstring(L, s);
  free(s);
  return 1;
}

////////////////////////////////
static int gmp_znum(lua_State *L) {
  _GMP_ZCHK_(z, 1)
  double d = mpz_get_d(*z);
  lua_pushnumber(L, d);
  return 1;
}

static int gmp_zadd(lua_State *L) {
  _GMP_ZCHK_ab
  mpz_add(*a, *a, *b);
  _GMP_ZLIMIT(*a);
  lua_pushvalue(L, 1);
  return 1;
}

static int gmp_zsub(lua_State *L) {
  _GMP_ZCHK_ab
  mpz_sub(*a, *a, *b);
  _GMP_ZLIMIT(*a);
  lua_pushvalue(L, 1);
  return 1;
}

static int gmp_zmul(lua_State *L) {
  _GMP_ZCHK_ab
  mpz_mul(*a, *a, *b);
  _GMP_ZLIMIT(*a);
  lua_pushvalue(L, 1);
  return 1;
}

static int gmp_zneg(lua_State *L) {
  _GMP_ZCHK_(z, 1)
  mpz_neg(*z, *z);
  lua_pushvalue(L, 1);
  return 1;
}

static int gmp_zabs(lua_State *L) {
  _GMP_ZCHK_(z, 1)
  mpz_abs(*z, *z);
  lua_pushvalue(L, 1);
  return 1;
}

static int gmp_zsgn(lua_State *L) {
  _GMP_ZCHK_(z, 1)
  int r = mpz_sgn(*z);
  lua_pushnumber(L, r);
  return 1;
}

static int gmp_zodd(lua_State *L) {
  _GMP_ZCHK_(z, 1)
  int r = mpz_odd_p(*z);
  lua_pushnumber(L, r);
  return 1;
}

static int gmp_zeven(lua_State *L) {
  _GMP_ZCHK_(z, 1)
  int r = mpz_even_p(*z);
  lua_pushnumber(L, r);
  return 1;
}

static int gmp_zaddmul(lua_State *L) {
  _GMP_ZCHK_zab
  mpz_addmul(*z, *a, *b);
  _GMP_ZLIMIT(*z);
  lua_pushvalue(L, 1);
  return 1;
}

static int gmp_zsubmul(lua_State *L) {
  _GMP_ZCHK_zab
  mpz_submul(*z, *a, *b);
  _GMP_ZLIMIT(*z);
  lua_pushvalue(L, 1);
  return 1;
}

////////////////////////////////
static int gmp_zdivtq(lua_State *L) {
  _GMP_ZCHK_ab
  _GMP_ZCHK_DIV_zero(b, "gmp_zdivtq() zero divisor")
  mpz_tdiv_q(*a, *a, *b);
  lua_pushvalue(L, 1);
  return 1;
}

////////////////////////////////
static int gmp_zdivtr(lua_State *L) {
  _GMP_ZCHK_ab
  _GMP_ZCHK_DIV_zero(b, "gmp_zdivtr() zero divisor")
  mpz_tdiv_r(*a, *a, *b);
  lua_pushvalue(L, 1);
  return 1;
}

////////////////////////////////
static int gmp_zdivfq(lua_State *L) {
  _GMP_ZCHK_ab
  _GMP_ZCHK_DIV_zero(b, "gmp_zdivfq() zero divisor")
  mpz_fdiv_q(*a, *a, *b);
  lua_pushvalue(L, 1);
  return 1;
}

////////////////////////////////
static int gmp_zdivfr(lua_State *L) {
  _GMP_ZCHK_ab
  _GMP_ZCHK_DIV_zero(b, "gmp_zdivfr() zero divisor")
  mpz_fdiv_r(*a, *a, *b);
  lua_pushvalue(L, 1);
  return 1;
}

////////////////////////////////
static int gmp_zdivcq(lua_State *L) {
  _GMP_ZCHK_ab
  _GMP_ZCHK_DIV_zero(b, "gmp_zdivcq() zero divisor")
  mpz_cdiv_q(*a, *a, *b);
  lua_pushvalue(L, 1);
  return 1;
}

////////////////////////////////
static int gmp_zdivcr(lua_State *L) {
  _GMP_ZCHK_ab
  _GMP_ZCHK_DIV_zero(b, "gmp_zdivcr() zero divisor")
  mpz_cdiv_r(*a, *a, *b);
  lua_pushvalue(L, 1);
  return 1;
}

////////////////////////////////
static int gmp_zmod(lua_State *L) {
  _GMP_ZCHK_ab
  _GMP_ZCHK_DIV_zero(b, "gmp_zmod() zero divisor")
  mpz_mod(*a, *a, *b);
  lua_pushvalue(L, 1);
  return 1;
}

////////////////////////////////
static int gmp_zcmp(lua_State *L) {
  _GMP_ZCHK_(a, 1)
  int r;
  if (lua_type(L, 2)==LUA_TNUMBER) {
    double b = luaL_checknumber(L, 2);
    r = mpz_cmp_d(*a, b);
  } else {
    _GMP_ZCHK_(b, 2)
    r = mpz_cmp(*a, *b);
  }
  lua_pushnumber(L, r);
  return 1;
}

////////////////////////////////
static int gmp_zcmpabs(lua_State *L) {
  _GMP_ZCHK_(a, 1)
  int r;
  if (lua_type(L, 2)==LUA_TNUMBER) {
    double b = luaL_checknumber(L, 2);
    r = mpz_cmpabs_d(*a, b);
  } else {
    _GMP_ZCHK_(b, 2)
    r = mpz_cmpabs(*a, *b);
  }
  lua_pushnumber(L, r);
  return 1;
}

////////////////////////////////
static int gmp_zand(lua_State *L) {
  _GMP_ZCHK_ab
  mpz_and(*a, *a, *b);
  lua_pushvalue(L, 1);
  return 1;
}

////////////////////////////////
static int gmp_zor(lua_State *L) {
  _GMP_ZCHK_ab
  mpz_ior(*a, *a, *b);
  lua_pushvalue(L, 1);
  return 1;
}

////////////////////////////////
static int gmp_zxor(lua_State *L) {
  _GMP_ZCHK_ab
  mpz_xor(*a, *a, *b);
  lua_pushvalue(L, 1);
  return 1;
}

////////////////////////////////
static int gmp_zcom(lua_State *L) {
  _GMP_ZCHK_ab
  mpz_com(*a, *b);
  lua_pushvalue(L, 1);
  return 1;
}

////////////////////////////////
static int gmp_zlshift(lua_State *L) {
  _GMP_ZCHK_(a, 1)
  unsigned long b = luaL_checklong(L, 2);
  _GMP_ZCHK_limit(b, "gmp_zlshift() limited")
  mpz_mul_2exp(*a, *a, b);
  _GMP_ZLIMIT(*a);
  lua_pushvalue(L, 1);
  return 1;
}

////////////////////////////////
static int gmp_zrshift(lua_State *L) {
  _GMP_ZCHK_(a, 1)
  unsigned long b = luaL_checklong(L, 2);
  mpz_tdiv_q_2exp(*a, *a, b);
  lua_pushvalue(L, 1);
  return 1;
}

static int gmp_zbset(lua_State *L) {
  _GMP_ZCHK_(a, 1)
  unsigned long b = luaL_checklong(L, 2);
  _GMP_ZCHK_limit(b, "gmp_zbset() limited")
  mpz_setbit(*a, b);
  lua_pushvalue(L, 1);
  return 1;
}

static int gmp_zbclr(lua_State *L) {
  _GMP_ZCHK_(a, 1)
  unsigned long b = luaL_checklong(L, 2);
  mpz_clrbit(*a, b);
  lua_pushvalue(L, 1);
  return 1;
}

static int gmp_zbtst(lua_State *L) {
  _GMP_ZCHK_(a, 1)
  unsigned long b = luaL_checklong(L, 2);
  lua_pushboolean(L, mpz_tstbit(*a,b));
  return 1;
}

////////////////////////////////
static int gmp_zswap(lua_State *L) {
  _GMP_ZCHK_ab
  mpz_swap(*a, *b);
  lua_pushvalue(L, 1);
  return 1;
}

////////////////////////////////
static const luaL_Reg gmp_zfuncs[] = {
  {"__gc", gmp_zclear},
  {"new", gmp_znew},
  {"set", gmp_zset},
  {"str", gmp_zstr},
  {"__tostring", gmp_zstr},
  {"num", gmp_znum},
  {"add", gmp_zadd},
  {"sub", gmp_zsub},
  {"mul", gmp_zmul},
  {"neg", gmp_zneg},
  {"abs", gmp_zabs},
  {"sgn", gmp_zsgn},
  {"odd", gmp_zodd},
  {"even", gmp_zeven},
  {"addmul", gmp_zaddmul},
  {"submul", gmp_zsubmul},
  {"div", gmp_zdivtq},
  {"rem", gmp_zdivtr},
  {"divtq", gmp_zdivtq},
  {"divtr", gmp_zdivtr},
  {"divfq", gmp_zdivfq},
  {"divfr", gmp_zdivfr},
  {"divcq", gmp_zdivcq},
  {"divcr", gmp_zdivcr},
  {"mod", gmp_zmod},
  {"cmp", gmp_zcmp},
  {"cmpabs", gmp_zcmpabs},
  {"band", gmp_zand},
  {"bor", gmp_zor},
  {"bxor", gmp_zxor},
  {"bcom", gmp_zcom},
  {"lshift", gmp_zlshift},
  {"rshift", gmp_zrshift},
  {"bset", gmp_zbset},
  {"bclr", gmp_zbclr},
  {"btst", gmp_zbtst},
  {"swap", gmp_zswap},
  {NULL, NULL}
};

////////////////////////////////
LUALIB_API int luaopen_gmp(lua_State *L, int zlimit) {
  char *limit;
  if (zlimit==128) {
    limit = "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF";
  } else if (zlimit==256) {
    limit = "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF";
  } else if (zlimit==512) {
    limit = "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF";
  } else {
    zlimit = 0;
  }
  if (zlimit && _gmp_zlimit==NULL) {
    _gmp_nlimit = zlimit;
    _gmp_zlimit = (mpz_t*)malloc(sizeof(mpz_t));
    mpz_init(*_gmp_zlimit);
    mpz_set_str(*_gmp_zlimit, limit, 16);
  }
  luaL_newmetatable(L, _GMP_ZMT);
  luaL_setfuncs(L, gmp_zfuncs, 0);
  lua_pushstring(L, "__index");
  lua_pushvalue(L, -2);
  lua_settable(L, -3);
  lua_setglobal(L, "mpz");
  return 1;
}
